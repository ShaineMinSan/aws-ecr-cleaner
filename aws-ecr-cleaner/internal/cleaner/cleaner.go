// aws-ecr-cleaner/internal/cleaner/cleaner.go
package cleaner

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"aws-ecr-cleaner/internal/config"
	"aws-ecr-cleaner/internal/ecr"
	"aws-ecr-cleaner/internal/k8s"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	awsecr "github.com/aws/aws-sdk-go/service/ecr"
)

func Run(cfg *config.Config) {
	log.Println("Starting AWS ECR Cleaner...")

	// 建立 AWS session
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(cfg.AWSRegion),
	})
	if err != nil {
		log.Fatalf("Failed to create AWS session: %v", err)
	}

	// 获取账户 ID并构造目标 ECR 地址
	accountID, err := ecr.GetAccountID(sess)
	if err != nil {
		log.Fatalf("Failed to get AWS account ID: %v", err)
	}
	targetECR := fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com", accountID, cfg.AWSRegion)
	fmt.Printf("Target ECR: %s\n", targetECR)

	// 获取 in-use 镜像映射（如果 imageListFile 不存在或为空，则从 k8s 集群拉取）
	var inUse map[string]bool
	fileInfo, err := os.Stat(cfg.ImageListFile)
	if err != nil || fileInfo.Size() == 0 {
		inUse = k8s.FetchInUseImages(cfg.ImageListFile)
	} else {
		inUse = k8s.LoadInUseImages(cfg.ImageListFile)
	}
	if cfg.Debug {
		log.Printf("[DEBUG] Loaded in-use images: %v", inUse)
	}

	// 创建 ECR 客户端
	svc := awsecr.New(sess)
	repos, err := ecr.GetRepositories(svc, cfg.TargetRepoRegex, cfg.Debug)
	if err != nil {
		log.Fatalf("Error fetching repositories: %v", err)
	}
	if len(repos) == 0 {
		fmt.Println("No repositories found matching the provided pattern.")
		os.Exit(0)
	}

	// 如果设置 EXCLUDE_REPO_REGEX，则过滤掉匹配的仓库
	var filteredRepos []*awsecr.Repository
	if cfg.ExcludeRepoRegex != "" {
		for _, repo := range repos {
			repoName := aws.StringValue(repo.RepositoryName)
			if ecr.MultiRegexMatch(repoName, cfg.ExcludeRepoRegex) {
				if cfg.Debug {
					log.Printf("[DEBUG] Excluding repository: %s", repoName)
				}
				continue
			}
			filteredRepos = append(filteredRepos, repo)
		}
		repos = filteredRepos
	}

	var scannedImages []ecr.ScannedImage
	var candidateImages []ecr.Candidate

	// 遍历每个仓库
	for _, repo := range repos {
		repoName := aws.StringValue(repo.RepositoryName)
		repoUri := aws.StringValue(repo.RepositoryUri)
		if !strings.HasPrefix(repoUri, targetECR) {
			continue
		}
		fmt.Printf("\nRepository: %s (URI: %s)\n", repoName, repoUri)

		images, err := ecr.GetImages(svc, repoName, cfg.Debug)
		if err != nil {
			fmt.Printf("Error fetching images for repository %s: %v\n", repoName, err)
			continue
		}
		fmt.Printf("Total images found: %d\n", len(images))

		// 如果仓库为空，则直接删除该仓库
		if len(images) == 0 {
			fmt.Printf("Repository %s is empty. Deleting repository.\n", repoName)
			if cfg.DryRun {
				fmt.Printf("[Dry-run] Would delete repository: %s\n", repoName)
			} else {
				delRepoInput := &awsecr.DeleteRepositoryInput{
					RepositoryName: aws.String(repoName),
					Force:          aws.Bool(true),
				}
				_, err := svc.DeleteRepository(delRepoInput)
				if err != nil {
					fmt.Printf("Error deleting repository %s: %v\n", repoName, err)
				} else {
					fmt.Printf("Deleted repository: %s\n", repoName)
				}
			}
			continue
		}

		// 记录扫描结果
		for _, image := range images {
			var tags []string
			if image.ImageTags != nil {
				for _, t := range image.ImageTags {
					tags = append(tags, aws.StringValue(t))
				}
			}
			var pushTime string
			if image.ImagePushedAt != nil {
				pushTime = image.ImagePushedAt.Format("2006-01-02T15:04:05Z")
			}
			scannedImages = append(scannedImages, ecr.ScannedImage{
				RepositoryName: repoName,
				RepositoryUri:  repoUri,
				ImageDigest:    aws.StringValue(image.ImageDigest),
				ImageTags:      tags,
				PushTime:       *image.ImagePushedAt,
			})
			fmt.Printf("  [Scanned] Digest: %s, Tags: %v, PushedAt: %s\n", aws.StringValue(image.ImageDigest), tags, pushTime)
		}

		// 根据规则过滤候选镜像
		candidates := ecr.FilterImagesForDeletion(images, cfg.HoldTagRegex, cfg.ProtectLatest, inUse, repoUri, cfg.ProtectInUseByK8s, cfg.Debug)
		if len(candidates) > 0 {
			fmt.Printf("\nCandidate images for deletion in repository '%s':\n", repoName)
			for _, cand := range candidates {
				cand.RepositoryName = repoName
				fmt.Printf("  [Candidate] Tag: %s, Digest: %s, PushedAt: %s\n", cand.ImageTag, cand.ImageDigest, cand.PushTime.Format("2006-01-02T15:04:05Z"))
				candidateImages = append(candidateImages, cand)
			}
		} else {
			fmt.Printf("No candidate images for deletion in repository '%s'.\n", repoName)
		}
	}

	// 打印扫描与候选列表
	fmt.Println("\n-------------------------------")
	fmt.Println("Original ECR Scanning Image List:")
	for _, s := range scannedImages {
		fmt.Printf("Repository: %s, Tags: %v, Digest: %s, PushedAt: %s\n", s.RepositoryName, s.ImageTags, s.ImageDigest, s.PushTime.Format("2006-01-02T15:04:05Z"))
	}

	fmt.Println("\nAfter Filter Image List (Candidates for deletion):")
	for _, c := range candidateImages {
		fmt.Printf("Repository: %s, Tag: %s, Digest: %s, PushedAt: %s\n", c.RepositoryName, c.ImageTag, c.ImageDigest, c.PushTime.Format("2006-01-02T15:04:05Z"))
	}
	fmt.Println("-------------------------------")

	if cfg.ListOnly {
		fmt.Println("\nList-only mode enabled. Exiting without deletion.")
		os.Exit(0)
	}

	// 如果不是自动确认模式，则进行交互确认
	if !cfg.AutoConfirm {
		fmt.Print("\nProceed with deletion of the above images? (y/n): ")
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			log.Fatalf("Failed to read input: %v", err)
		}
		input = strings.TrimSpace(input)
		if strings.ToLower(input) != "y" {
			fmt.Println("Aborting deletion.")
			os.Exit(0)
		}
	}

	// 遍历候选镜像执行删除
	for _, cand := range candidateImages {
		err := ecr.DeleteImage(svc, cand, cfg.DryRun, cfg.Debug)
		if err != nil {
			fmt.Printf("Error deleting image (Tag %s) in repository %s: %v\n", cand.ImageTag, cand.RepositoryName, err)
		}
	}

	// 删除候选镜像后，再次检查每个仓库是否为空，若空则删除仓库（使用 Force 删除）
	fmt.Println("\n-------------------------------")
	fmt.Println("Checking for empty repositories to delete...")
	for _, repo := range repos {
		repoName := aws.StringValue(repo.RepositoryName)
		remainingImages, err := ecr.GetImages(svc, repoName, cfg.Debug)
		if err != nil {
			fmt.Printf("Error re-fetching images for repository %s: %v\n", repoName, err)
			continue
		}
		if len(remainingImages) == 0 {
			fmt.Printf("Repository %s is now empty. Deleting repository.\n", repoName)
			if cfg.DryRun {
				fmt.Printf("[Dry-run] Would delete repository: %s\n", repoName)
			} else {
				delRepoInput := &awsecr.DeleteRepositoryInput{
					RepositoryName: aws.String(repoName),
					Force:          aws.Bool(true),
				}
				_, err := svc.DeleteRepository(delRepoInput)
				if err != nil {
					fmt.Printf("Error deleting repository %s: %v\n", repoName, err)
				} else {
					fmt.Printf("Deleted repository: %s\n", repoName)
				}
			}
		}
	}

	fmt.Println("\n-------------------------------")
	fmt.Println("Deletion process completed.")
}
