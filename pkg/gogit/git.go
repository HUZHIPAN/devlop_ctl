package gogit

import (
	"fmt"
	"lwapp/pkg/diary"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
	"github.com/go-git/go-git/v5/plumbing/object"
)

var openTimeCost = false // 开启耗时统计

//@brief：耗时统计函数
func timeCost(identification string) func() {
	start := time.Now()
	return func() {
		tc := time.Since(start)
		fmt.Printf("%v ：time cost = %v\n", identification, tc)
	}
}

// 目录是否一个git仓库
func IsRepository(dirPath string) bool {
	r := openRepository(dirPath)
	return r != nil
}

// 仓库工作区改动列表
func RepositoryWorkSpaceStatus(dirPath string, excludes []string) (git.Status, error) {
	if openTimeCost {
		defer timeCost("获取仓库变动列表" + dirPath)()
	}

	workTree := openRepositoryGetWorkTree(dirPath)
	if workTree == nil {
		return nil, fmt.Errorf("无法打开目标仓库工作区:（%v）", dirPath)
	}

	excludePatterns := []gitignore.Pattern{}
	for _, excludeRow := range excludes {
		excludePatterns = append(excludePatterns, gitignore.ParsePattern(excludeRow, []string{}))
	}

	workTree.Excludes = excludePatterns
	s, err := workTree.Status()
	if err != nil {
		diary.Errorf("获取仓库（%v）工作区status失败：%v", dirPath, err)
		return nil, err
	}
	return s, nil

}

// 丢弃仓库工作区未提交的变动
func CleanRepositoryWorkspaceChange(dirPath string, excludes []string) bool {
	if openTimeCost {
		defer timeCost("丢弃仓库工作区未提交的变动" + dirPath)()
	}

	workTree := openRepositoryGetWorkTree(dirPath)
	if workTree == nil {
		return false
	}

	excludePatterns := []gitignore.Pattern{}
	for _, excludeRow := range excludes {
		excludePatterns = append(excludePatterns, gitignore.ParsePattern(excludeRow, []string{}))
	}

	workTree.Excludes = excludePatterns

	err := workTree.Clean(&git.CleanOptions{Dir: true})
	if err != nil {
		diary.Errorf("清除仓库（%v）工作区未追踪文件失败：%v", dirPath, err)
		return false
	}

	return true
}

// 获取仓库当前所处分支
func GetRepositoryCurrentBranch(dirPath string) string {
	if openTimeCost {
		defer timeCost("获取仓库当前所处分支" + dirPath)()
	}

	repository := openRepository(dirPath)
	if repository == nil {
		return ""
	}

	head, err := repository.Head()
	if err != nil {
		diary.Errorf("获取仓库（%v）HEAD指针失败：%v", dirPath, err)
		return "master"
	}
	return strings.TrimPrefix(head.Name().String(), "refs/heads/")
}

// 获取仓库分支列表
func GetRepositoryBranchList(dirPath string) []string {
	if openTimeCost {
		defer timeCost("获取仓库分支列表" + dirPath)()
	}

	repository := openRepository(dirPath)
	if repository == nil {
		return []string{}
	}
	referenceIter, err := repository.References()
	if err != nil {
		diary.Errorf("获取仓库（%v）分支列表失败: %v", dirPath, err)
		return []string{}
	}
	branchList := []string{}
	referenceIter.ForEach(func(r *plumbing.Reference) error {
		branchName := strings.TrimPrefix(r.Name().String(), "refs/heads/")
		if branchName != "HEAD" {
			branchList = append(branchList, branchName)
		}
		return nil
	})
	return branchList
}

// 基于当前head创建一个分支
func CreateRepositoryFromByCurrentHead(dirPath string, branchName string) bool {
	if openTimeCost {
		defer timeCost("基于当前head创建分支" + dirPath)()
	}

	repository := openRepository(dirPath)
	if repository == nil {
		return false
	}

	if branchName == GetRepositoryCurrentBranch(dirPath) {
		diary.Errorf("仓库（%v）分支（%v）已经存在！", dirPath, branchName)
		return false
	}

	headRef, err := repository.Head()
	if err != nil {
		diary.Errorf("仓库（%v）获取head引用失败：%v", dirPath, err)
	}

	return CreateRepositoryBranchFromRef(dirPath, headRef, branchName)
}

// 获取当前HEAD引用hash
func GetRepositoryCurrentHeadRefHash(dirPath string) string {
	if openTimeCost {
		defer timeCost("获取当前head引用hash值" + dirPath)()
	}

	repository := openRepository(dirPath)
	if repository == nil {
		return "HEAD"
	}

	headRef, err := repository.Head()
	if err != nil {
		diary.Errorf("仓库（%v）获取head引用失败：%v", dirPath, err)
		return "HEAD"
	}

	return headRef.Hash().String()
}

// 获取分支引用
func GetRepositoryBranchRef(dirPath string, branchName string) *plumbing.Reference {
	if openTimeCost {
		defer timeCost("获取分支引用" + dirPath)()
	}
	repository := openRepository(dirPath)
	if repository == nil {
		return nil
	}

	referenceName := plumbing.NewBranchReferenceName(branchName)

	ref, err := repository.Reference(referenceName, false)
	if err != nil {
		diary.Errorf("仓库（%v）获取分支（%v）引用失败：%v", dirPath, branchName, err)
		return nil
	}

	return ref
}

// 基于master分支创建一个分支
func CreateRepositoryBranchFromByMaster(dirPath string, branchName string) bool {
	if openTimeCost {
		defer timeCost("基于master分支创建一个分支" + dirPath)()
	}
	repository := openRepository(dirPath)
	if repository == nil {
		return false
	}

	masterRef, err := repository.Reference(plumbing.Master, false)
	if err != nil {
		diary.Errorf("仓库（%v）获取master分支引用失败：%v", dirPath, err)
	}

	return CreateRepositoryBranchFromRef(dirPath, masterRef, branchName)
}

// 基于某个提交引用创建一个分支
func CreateRepositoryBranchFromRef(dirPath string, ref *plumbing.Reference, branchName string) bool {
	if openTimeCost {
		defer timeCost("基于某个提交创建一个分支" + dirPath)()
	}
	repository := openRepository(dirPath)
	if repository == nil {
		return false
	}

	newBranchRef := plumbing.NewHashReference(plumbing.NewBranchReferenceName(branchName), ref.Hash())
	err := repository.Storer.SetReference(newBranchRef)
	if err != nil {
		diary.Errorf("仓库（%v）创建分支（%v）失败：%v", dirPath, branchName, err)
		return false
	}

	return true
}

// 删除分支
func DeleteBranchName(dirPath string, branchName string) bool {
	if openTimeCost {
		defer timeCost("删除分支" + dirPath + ">>" + branchName)()
	}
	repository := openRepository(dirPath)
	if repository == nil {
		return false
	}
	err := repository.Storer.RemoveReference(plumbing.NewBranchReferenceName(branchName))
	if err != nil {
		diary.Errorf("仓库（%v）删除分支（%v）失败：%v", dirPath, branchName, err)
		return false
	}

	return true
}

// 检查一个分支是否存在
func CheckBranchIsExist(dirPath string, branchName string) bool {
	refName := plumbing.NewBranchReferenceName(branchName)

	repository := openRepository(dirPath)
	if repository == nil {
		return false
	}
	ref, err := repository.Reference(refName, false)
	if err != nil || ref == nil {
		return false
	}

	return true
}

// 切换仓库工作区到指定分支
func CheckoutRepositoryBranch(dirPath string, branchName string, excludes []string) bool {
	if openTimeCost {
		defer timeCost("切换分支" + dirPath + ">>" + branchName)()
	}

	workTree := openRepositoryGetWorkTree(dirPath)

	if workTree == nil {
		return false
	}

	excludePatterns := []gitignore.Pattern{}
	for _, excludeRow := range excludes {
		excludePatterns = append(excludePatterns, gitignore.ParsePattern(excludeRow, []string{}))
	}
	workTree.Excludes = excludePatterns

	opt := &git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(branchName),
		Force:  true,
		Keep:   false,
	}

	err := workTree.Checkout(opt)
	if err != nil {
		diary.Errorf("仓库（%v）切换到分支失败（%v）：%v", dirPath, branchName, err)
		return false
	}
	err = workTree.Clean(&git.CleanOptions{Dir: true})

	if err != nil {
		diary.Errorf("清除仓库（%v）工作区未追踪文件失败：%v", dirPath, err)
	}
	return true
}

// 打开一个git仓库
func openRepository(dirPath string) *git.Repository {
	if openTimeCost {
		defer timeCost("打开git仓库" + dirPath)()
	}
	repository, err := git.PlainOpen(dirPath)
	if err != nil {
		diary.Errorf("打开git仓库（%v）失败：%v", dirPath, err)
		return nil
	}

	return repository
}

// 打开一个git仓库并获取工作区
func openRepositoryGetWorkTree(dirPath string) *git.Worktree {
	if openTimeCost {
		defer timeCost("打开git仓库获取工作区" + dirPath)()
	}
	repository := openRepository(dirPath)
	if repository == nil {
		return nil
	}

	workTree, err := repository.Worktree()
	if err != nil {
		diary.Errorf("打开仓库（%v）工作区失败：%v", dirPath, err)
		return nil
	}
	return workTree
}

// 获取最近几次commit的hash
func getRepositoryCommitLogHash(repository *git.Repository, num int) (plumbing.Hash, error) {
	opt := &git.LogOptions{}
	commitIter, err := repository.Log(opt)
	if err != nil {
		diary.Errorf("获取仓库commit log hash失败：%v", err)
	}
	for i := 1; i <= num; i++ {
		c, err := commitIter.Next()
		if err != nil {
			diary.Errorf("获取最近%v次hash失败：%v", i, err)
			return plumbing.Hash{}, err
		}
		if i == num {
			return c.Hash, nil
		}
	}
	return plumbing.Hash{}, fmt.Errorf("没有找到对应提交记录")
}

// 初始化目录的版本控制
func InitializeDirVersionControl(dirPath string) bool {
	if openTimeCost {
		defer timeCost("初始化目录版本控制" + dirPath)()
	}

	_, err := git.PlainInit(dirPath, false)
	if err != nil {
		diary.Errorf("初始化目录（%v）版本控制失败：%v", dirPath, err)
		return false
	}

	return true
}

// 提交指定仓库目录的代码
func CommitDirChange(dirPath string, message string, excludes []string) bool {
	if openTimeCost {
		defer timeCost("提交目录代码" + dirPath)()
	}
	excludePatterns := []gitignore.Pattern{}
	for _, excludeRow := range excludes {
		excludePatterns = append(excludePatterns, gitignore.ParsePattern(excludeRow, []string{}))
	}

	workTree := openRepositoryGetWorkTree(dirPath)
	if workTree == nil {
		return false
	}

	workTree.Excludes = excludePatterns
	err := workTree.AddWithOptions(&git.AddOptions{All: true})
	if err != nil {
		diary.Errorf("仓库（%v）添加未提交变动到版本管理失败（git add .）: %v", dirPath, err)
		return false
	}

	author := &object.Signature{
		Name:  "lwctl",
		Email: "lewei@lwops.cn",
		When:  time.Now(),
	}
	_, err = workTree.Commit(message, &git.CommitOptions{All: true, Author: author})
	if err != nil {
		diary.Errorf("仓库（%v）工作区提交失败: %v", dirPath, err)
		return false
	}
	return true
}

// 获取某个分支的commit历史
func GetBranchCommitInfo(dirPath string, branchName string) []map[string]string {
	if openTimeCost {
		defer timeCost("获取分支提交历史" + dirPath)()
	}
	repository := openRepository(dirPath)
	if repository == nil {
		return []map[string]string{}
	}

	branchRef, err := repository.Reference(plumbing.NewBranchReferenceName(branchName), false)

	if err != nil {
		diary.Errorf("仓库（%v）获取分支（%v）引用失败：%v", dirPath, branchName, err)
		return []map[string]string{}
	}

	opt := &git.LogOptions{
		From: branchRef.Hash(),
	}
	commitIter, err := repository.Log(opt)
	if err != nil {
		return []map[string]string{}
	}

	branchCommitInfo := []map[string]string{}
	commitIter.ForEach(func(c *object.Commit) error {
		branchCommitInfo = append(branchCommitInfo, map[string]string{
			"time":    c.Author.When.Format("2006-01-02 15:04:05"),
			"hash":    c.Hash.String(),
			"message": c.Message,
		})
		return nil
	})

	return branchCommitInfo
}

// 获取一个提交的变更列表
func GetCommitChangeList(dirPath, commitHash string) (*[]map[string]string, error) {
	if openTimeCost {
		defer timeCost("获取一个提交的变更列表" + dirPath)()
	}
	registry := openRepository(dirPath)
	if registry == nil {
		return nil, fmt.Errorf("打开仓库异常")
	}

	hash := plumbing.NewHash(commitHash)
	commitObject, err := object.GetCommit(registry.Storer, hash)
	if err != nil {
		diary.Errorf("获取仓库（%v）的commit（%v）失败 ：%v", dirPath, commitHash, err)
		return nil, err
	}

	if commitObject.NumParents() <= 0 { // 没有上一级提交
		return &[]map[string]string{}, nil
	}

	parentCommitObject, err := commitObject.Parent(0)
	if err != nil {
		diary.Errorf("获取仓库（%v）的commit（%v）的上一级提交失败 ：%v", dirPath, commitHash, err)
		return nil, err
	}

	// 当前提交与它的上一个提交的差异，所以to是更新前，from是更新后
	patch, err := commitObject.Patch(parentCommitObject)
	if err != nil {
		diary.Errorf("获取仓库（%v）的commit（%v）的文件变动列表失败 ：%v", dirPath, commitHash, err)
		return nil, err
	}
	fileDiff := patch.FilePatches()

	var (
		toPath   string
		fromPath string
	)
	changeList := &[]map[string]string{}
	var changeItem map[string]string
	for _, filePath := range fileDiff {
		from, to := filePath.Files()
		if to == nil {
			changeItem = map[string]string{
				"file": from.Path(),
				"mode": "新增",
			}
		} else if from == nil {
			changeItem = map[string]string{
				"file": to.Path(),
				"mode": "删除",
			}
		} else if from != nil && to != nil {
			changeItem = map[string]string{
				"file": from.Path(),
				"mode": "修改",
			}
		} else {
			if from != nil {
				fromPath = from.Path()
			} else {
				fromPath = ""
			}
			if to != nil {
				toPath = to.Path()
			} else {
				toPath = ""
			}
			changeItem = map[string]string{
				"file": fmt.Sprintf("%v => %v", toPath, fromPath),
				"mode": "",
			}
		}
		*changeList = append(*changeList, changeItem)
	}

	return changeList, nil
}

// 回滚一个目录指定次数提交
func RollbackDirChangeSpan(dirPath string, lastCommitNum int) bool {
	if openTimeCost {
		defer timeCost("回滚一个目录指定次数提交" + dirPath)()
	}
	repository := openRepository(dirPath)
	if repository == nil {
		return false
	}
	commitHash, err := getRepositoryCommitLogHash(repository, lastCommitNum)
	if err != nil {
		diary.Errorf("获取仓库（%v）最近%v次提交hash失败 ：%v", dirPath, lastCommitNum, err)
		return false
	}
	opt := &git.ResetOptions{
		Commit: commitHash,
		Mode:   git.HardReset,
	}
	workTree, err := repository.Worktree()
	if err != nil {
		return false
	}

	err = workTree.Reset(opt)
	return err == nil
}

// git仓库目录是否初始化提交过
func IsDirVersionInitialized(dirPath string) bool {
	repository := openRepository(dirPath)
	if repository == nil {
		return false
	}

	_, err := repository.Head()
	if err != nil {
		return false
	}
	return true
}
