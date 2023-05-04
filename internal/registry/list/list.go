// Tools to list and print images in the Jetstack Enterprise Registry
package list

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"

	"github.com/Masterminds/semver"
	"github.com/google/go-containerregistry/pkg/crane"
	"golang.org/x/sync/errgroup"
)

type versions []*semver.Version

func (o versions) Latest() *semver.Version {
	if len(o) == 0 {
		return nil
	}
	sort.Sort(sort.Reverse(semver.Collection(o)))
	for _, v := range o {
		if v.Prerelease() == "" {
			return v
		}
	}
	return nil
}

func (o versions) String() string {
	out := make([]string, len(o))
	sort.Sort(sort.Reverse(semver.Collection(o)))
	for i, v := range o {
		out[i] = v.String()
	}
	return strings.Join(out, ", ")
}

// listVersions returns the semver tags of the given image in the OCI repository.
func listVersions(ctx context.Context, image string) (versions, error) {
	listing, err := crane.ListTags(image, crane.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	var versions versions
	for _, item := range listing {
		item = strings.TrimSpace(item)
		if len(item) == 0 {
			continue
		}
		v, err := semver.NewVersion(item)
		if err != nil {
			log.Printf("Ignoring non-semver tag %q for image %q", item, image)
			continue
		}
		versions = append(versions, v)
	}
	return versions, nil
}

// listImages returns a list of all the images in the repository
func listImages(ctx context.Context, repository string, filter string) ([]string, error) {
	cmd := exec.CommandContext(ctx, "gcloud", "container", "images", "list", "--format=value(name)", "--repository", repository, "--filter", filter)
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var images []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		images = append(images, line)
	}
	return images, nil
}

// Print a markdown table of images and their latest version tags.
// Images without any semver tags or without a stable version are omitted.
func Print(ctx context.Context, repository, imagesFilter string) {
	images, err := listImages(ctx, repository, imagesFilter)
	if err != nil {
		log.Fatal(err)
	}
	m := sync.Map{}
	g, gCTX := errgroup.WithContext(ctx)
	g.SetLimit(5)
	for _, image := range images {
		image := image
		g.Go(func() error {
			versions, err := listVersions(gCTX, image)
			if err != nil {
				return err
			}
			if latestVersion := versions.Latest(); latestVersion != nil {
				m.Store(image, latestVersion)
			} else {
				log.Printf("Ignoring image %q without stable semver version (%s)", image, versions.String())
			}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		log.Fatal(err)
	}
	output := []string{}
	m.Range(func(image any, latestVersion any) bool {
		output = append(output, fmt.Sprintf("| `%s` | `v%s` |", image, latestVersion))
		return true
	})
	sort.Sort(sort.StringSlice(output))
	output = append([]string{
		"| Image | Tag |",
		"|-------|-----|",
	}, output...)
	fmt.Println(strings.Join(output, "\n"))
}
