package gogit

import (
	"fmt"
	"testing"
)

func TestInitializeDirVersionControl(t *testing.T) {
	InitializeDirVersionControl("../../demo")

	fmt.Println(t)
}

func TestCommitDirChange(t *testing.T) {
	// excludes := []gitignore.Pattern{
	// 	gitignore.ParsePattern("125.*", []string{}),
	// 	gitignore.ParsePattern("package1", []string{}),
	// }

	// Demo("../../demo", "测试commit0001", excludes)

	CommitDirChange("../../demo", "测试commit0001", []string{})
}

func TestRollbackDirChangeSpan(t *testing.T) {

	RollbackDirChangeSpan("../../demo", 4)
}

func TestGetRepositoryCurrentBranch(t *testing.T) {
	branchName := GetRepositoryCurrentBranch("../../demo")

	fmt.Println(branchName)
}

func TestCreateRepositoryFromByMaster(t *testing.T) {
	CreateRepositoryBranchFromByMaster("../../demo", "branch2")
}

func TestCheckoutRepositoryBranch(t *testing.T) {
	CheckoutRepositoryBranch("../../demo", "branch2", []string{"1"})
}

func TestRepositoryWorkSpaceStatus(t *testing.T) {
	RepositoryWorkSpaceStatus("../../demo", []string{"nginx-", "pack"})
}

func TestCleanRepositoryWorkspaceChange(t *testing.T) {
	ok := CleanRepositoryWorkspaceChange("../../demo", []string{"/1", "/124.txt"})
	fmt.Println(ok)
}

func TestGetRepositoryBranchList(t *testing.T) {
	GetRepositoryBranchList("../../demo")
}

func TestCreateRepositoryFromByCurrentHead(t *testing.T) {
	CreateRepositoryFromByCurrentHead("../../demo", "branch3")
}

func TestGetBranchCommitInfo(t *testing.T) {
	GetBranchCommitInfo("../../demo", "branch4")
}

func TestGetRepositoryCurrentHeadRefHash(t *testing.T) {
	hash := GetRepositoryCurrentHeadRefHash("../../demo")

	fmt.Println(hash)
}

func TestGetRepositoryBranchRef(t *testing.T) {
	GetRepositoryBranchRef("../../demo", "branch4")
}

func TestGetCommitChangeList(t *testing.T) {
	GetCommitChangeList("../../demo", "d6b8a5fb11014b5f191dda2a911c069ad2ecd943")
}

func TestDeleteBranchName(t *testing.T) {
	DeleteBranchName("../../demo", "branch3")
}

func TestCheckBranchIsExist(t *testing.T) {
	CheckBranchIsExist("../../demo", "branch4")
}
