package packer

import (
	"fmt"
	"testing"
)

func TestUnpack(t *testing.T) {
	pack := NewTgzPacker()

	err := pack.UnPack("../../demo/feature_lwctl_increment_package.tar.gz", "../../demo", []string{"nginx-1.22.0/auto", "nginx-1.22.0/contrib", "nginx-1.22.0/html/"}, []string{})

	fmt.Println(err)
}

func TestPack(t *testing.T) {
	pack := NewTgzPacker()

	pack.Pack("../../demo/nginx-1.22.0", "../../demo/nignx_007.tar.gz")
	// pack.Pack("C:/Users/HUZHIPAN/Desktop/go/lwapp/demo/nginx-1.22.0", "../../demo/nginx_004.tar.gz")
}
