package docker

import (
	"fmt"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
)

func TestImages(t *testing.T) {
	client := NewDockerClient()

	opt := filters.NewArgs(filters.KeyValuePair{Key: "reference", Value: "nginx"})
	rows, err := client.Images(types.ImageListOptions{Filters: opt})

	fmt.Println(rows, err)
}

func TestRemoveDanglingImages(t *testing.T) {
	client := NewDockerClient()
	client.RemoveDanglingImages()
}

func TestLoadImage(t *testing.T) {
	imageFile := "./itops_v1_2_x86_64.tar"
	client := NewDockerClient()
	client.LoadImage(imageFile)
}

func TestRemoveImage(t *testing.T) {
	client := NewDockerClient()
	client.RemoveImage("itops:v1.2")
}

func TestReTagImage(t *testing.T) {
	client := NewDockerClient()
	client.ReTagImage("itops:v1.2", "lwops_image_1")
}

func TestGetImageIdByTag(t *testing.T) {
	client := NewDockerClient()
	client.GetImageIdByTag("lwapp_image_web:8")
}
