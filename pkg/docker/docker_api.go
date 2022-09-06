package docker

import (
	"context"
	"lwapp/pkg/diary"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

// 1.Docker docker client
type Docker struct {
	*client.Client
}

//  2.init docker client
func NewDockerClient() *Docker {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil
	}
	return &Docker{
		cli,
	}
}

// 根据image tag获取image ID
func (d *Docker) GetImageIdByTag(tagName string) string {
	args := filters.NewArgs(filters.KeyValuePair{Key: "reference", Value: tagName})
	opt := types.ImageListOptions{
		Filters: args,
	}
	imageList, err := d.ImageList(context.TODO(), opt)
	if err != nil {
		return ""
	}
	for _, img := range imageList {
		return img.ID
	}
	return ""
}

// Images get images from
func (d *Docker) Images(opt types.ImageListOptions) ([]types.ImageSummary, error) {
	return d.ImageList(context.TODO(), opt)
}

// LoadImage load image from tar file
func (d *Docker) LoadImage(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = d.ImageLoad(context.TODO(), file, true)
	return err
}

// 将镜像重命名tag
func (d *Docker) ReTagImage(souceTag string, targetTag string) error {
	err := d.ImageTag(context.TODO(), souceTag, targetTag)
	return err
}

// RemoveImage remove image 这里需要注意的一点就是移除了镜像之后，
//会出现<none>:<none>的标签，这个是因为下载的镜像是分层的，所以删除会导致
func (d *Docker) RemoveImage(name string) error {
	_, err := d.ImageRemove(context.TODO(), name, types.ImageRemoveOptions{})
	return err
}

//RemoveDanglingImages remove dangling images  <none>
func (d *Docker) RemoveDanglingImages() error {
	opt := types.ImageListOptions{
		Filters: filters.NewArgs(filters.Arg("dangling", "true")),
	}

	images, err := d.Images(opt)
	if err != nil {
		return err
	}

	errIDs := []string{}

	for _, image := range images {
		diary.Infof("try to delete <none> image.ID: %v", image.ID)
		if err := d.RemoveImage(image.ID); err != nil {
			errIDs = append(errIDs, image.ID[7:19])
		}
	}

	if len(errIDs) > 1 {
		diary.Warningf("can not remove ids: %s\n", errIDs)
	}

	return nil
}
