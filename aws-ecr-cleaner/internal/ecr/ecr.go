// aws-ecr-cleaner/internal/ecr/ecr.go
package ecr

import (
	"fmt"
	"log"
	"sort"
	"time"

	"aws-ecr-cleaner/internal/util"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/aws/aws-sdk-go/service/sts"
)

// Candidate 保存待删除的镜像信息
type Candidate struct {
	RepositoryName string
	RepositoryUri  string
	ImageDigest    string
	ImageTag       string
	PushTime       time.Time
}

// ScannedImage 保存扫描到的镜像信息
type ScannedImage struct {
	RepositoryName string
	RepositoryUri  string
	ImageDigest    string
	ImageTags      []string
	PushTime       time.Time
}

// GetRepositories 获取所有仓库，并用 compositeRegex 过滤
func GetRepositories(svc *ecr.ECR, compositeRegex string, debug bool) ([]*ecr.Repository, error) {
	var repos []*ecr.Repository
	input := &ecr.DescribeRepositoriesInput{}
	err := svc.DescribeRepositoriesPages(input, func(page *ecr.DescribeRepositoriesOutput, lastPage bool) bool {
		for _, repo := range page.Repositories {
			repoName := aws.StringValue(repo.RepositoryName)
			if util.CompositeMatch(repoName, compositeRegex) {
				repos = append(repos, repo)
				if debug {
					log.Printf("[DEBUG] Matched repository: %s", repoName)
				}
			}
		}
		return !lastPage
	})
	if err != nil {
		return nil, err
	}
	return repos, nil
}

// GetImages 获取指定仓库的所有镜像详情
func GetImages(svc *ecr.ECR, repositoryName string, debug bool) ([]*ecr.ImageDetail, error) {
	var images []*ecr.ImageDetail
	input := &ecr.DescribeImagesInput{
		RepositoryName: aws.String(repositoryName),
	}
	err := svc.DescribeImagesPages(input, func(page *ecr.DescribeImagesOutput, lastPage bool) bool {
		images = append(images, page.ImageDetails...)
		return !lastPage
	})
	if err != nil {
		return nil, err
	}
	if debug {
		log.Printf("[DEBUG] Found %d images in repository %s", len(images), repositoryName)
	}
	return images, nil
}

// FilterImagesForDeletion 根据规则过滤候选镜像
// 修改：若镜像未打标签，则直接加入候选删除列表
func FilterImagesForDeletion(images []*ecr.ImageDetail, holdTagRegex string, protectLatest int, inUse map[string]bool, repositoryUri string, protectInUse bool, debug bool) []Candidate {
	var inUseCandidates []Candidate
	var notInUseCandidates []Candidate

	trimmedRepoUri := util.TrimRegistry(repositoryUri)

	for _, image := range images {
		var pushTime time.Time
		if image.ImagePushedAt != nil {
			pushTime = *image.ImagePushedAt
		}
		// 若镜像未打标签，直接作为候选删除
		if image.ImageTags == nil || len(image.ImageTags) == 0 {
			cand := Candidate{
				RepositoryUri: repositoryUri,
				ImageDigest:   aws.StringValue(image.ImageDigest),
				ImageTag:      "", // untagged
				PushTime:      pushTime,
			}
			notInUseCandidates = append(notInUseCandidates, cand)
			continue
		}

		// 对有标签的镜像处理
		var tagList []string
		for _, tag := range image.ImageTags {
			tagList = append(tagList, aws.StringValue(tag))
		}
		combinedTags := fmt.Sprintf("%s", tagList)

		// 如果标签匹配 holdTagRegex，则跳过删除
		if util.HoldTagMatch(combinedTags, holdTagRegex) {
			if debug {
				log.Printf("[DEBUG] Holding image %s with tags: %s", trimmedRepoUri, combinedTags)
			}
			continue
		}

		used := false
		for _, tag := range image.ImageTags {
			tagVal := aws.StringValue(tag)
			fullImageUri := fmt.Sprintf("%s:%s", trimmedRepoUri, tagVal)
			if protectInUse && inUse[fullImageUri] {
				used = true
				break
			}
		}

		cand := Candidate{
			RepositoryUri: repositoryUri,
			ImageDigest:   aws.StringValue(image.ImageDigest),
			ImageTag:      aws.StringValue(image.ImageTags[0]),
			PushTime:      pushTime,
		}

		if used {
			inUseCandidates = append(inUseCandidates, cand)
		} else {
			notInUseCandidates = append(notInUseCandidates, cand)
		}
	}

	sort.Slice(inUseCandidates, func(i, j int) bool {
		return inUseCandidates[i].PushTime.After(inUseCandidates[j].PushTime)
	})
	protectedCount := protectLatest
	if protectedCount > len(inUseCandidates) {
		protectedCount = len(inUseCandidates)
	}
	if debug && protectedCount > 0 {
		log.Printf("[DEBUG] Protecting %d in-use images (newest)", protectedCount)
	}
	var inUseForDeletion []Candidate
	if len(inUseCandidates) > protectedCount {
		inUseForDeletion = inUseCandidates[protectedCount:]
	}

	candidates := append(notInUseCandidates, inUseForDeletion...)
	return candidates
}

// GetAccountID 调用 STS 获取 AWS 账户 ID
func GetAccountID(sess *session.Session) (string, error) {
	stsClient := sts.New(sess)
	input := &sts.GetCallerIdentityInput{}
	result, err := stsClient.GetCallerIdentity(input)
	if err != nil {
		return "", err
	}
	return aws.StringValue(result.Account), nil
}

// DeleteImage 删除候选镜像
func DeleteImage(svc *ecr.ECR, candidate Candidate, dryRun bool, debug bool) error {
	if dryRun {
		log.Printf("[Dry-run] Would delete image in repository %s: Tag %s, Digest %s", candidate.RepositoryName, candidate.ImageTag, candidate.ImageDigest)
		return nil
	}

	// 构造 ImageIdentifier，如果 ImageTag 为空，则不设置该字段
	imageIdentifier := &ecr.ImageIdentifier{
		ImageDigest: aws.String(candidate.ImageDigest),
	}
	if candidate.ImageTag != "" {
		imageIdentifier.ImageTag = aws.String(candidate.ImageTag)
	}

	delInput := &ecr.BatchDeleteImageInput{
		RepositoryName: aws.String(candidate.RepositoryName),
		ImageIds:       []*ecr.ImageIdentifier{imageIdentifier},
	}
	result, err := svc.BatchDeleteImage(delInput)
	if err != nil {
		return err
	}
	if len(result.Failures) > 0 {
		return fmt.Errorf("failed to delete image: %v", result.Failures)
	}
	if debug {
		log.Printf("[DEBUG] Deleted image response: %v", result)
	}
	log.Printf("Deleted image (Tag %s) in repository %s", candidate.ImageTag, candidate.RepositoryName)
	return nil
}

// MultiRegexMatch 暴露工具函数给外部使用（如在过滤仓库时使用 EXCLUDE_REPO_REGEX）
func MultiRegexMatch(s, config string) bool {
	return util.MultiRegexMatch(s, config)
}
