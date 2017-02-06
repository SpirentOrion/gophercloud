package images

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/pagination"
)

// Image model
// Does not include the literal image data; just metadata.
// returned by listing images, and by fetching a specific image.
type Image struct {
	// ID is the image UUID
	ID string `json:"id"`

	// Name is the human-readable display name for the image.
	Name string `json:"name"`

	// Status is the image status. It can be "queued" or "active"
	// See imageservice/v2/images/type.go
	Status ImageStatus `json:"status"`

	// Tags is a list of image tags. Tags are arbitrarily defined strings
	// attached to an image.
	Tags []string `json:"tags"`

	// ContainerFormat is the format of the container.
	// Valid values are ami, ari, aki, bare, and ovf.
	ContainerFormat string `json:"container_format"`

	// DiskFormat is the format of the disk.
	// If set, valid values are ami, ari, aki, vhd, vmdk, raw, qcow2, vdi, and iso.
	DiskFormat string `json:"disk_format"`

	// MinDiskGigabytes is the amount of disk space in GB that is required to boot the image.
	MinDiskGigabytes int `json:"min_disk"`

	// MinRAMMegabytes [optional] is the amount of RAM in MB that is required to boot the image.
	MinRAMMegabytes int `json:"min_ram"`

	// Owner is the tenant the image belongs to.
	Owner string `json:"owner"`

	// Protected is whether the image is deletable or not.
	Protected bool `json:"protected"`

	// Visibility defines who can see/use the image.
	Visibility ImageVisibility `json:"visibility"`

	// Checksum is the checksum of the data that's associated with the image
	Checksum string `json:"checksum"`

	// SizeBytes is the size of the data that's associated with the image.
	SizeBytes int64 `json:"size"`

	// Metadata is a set of metadata associated with the image.
	// Image metadata allow for meaningfully define the image properties
	// and tags. See http://docs.openstack.org/developer/glance/metadefs-concepts.html.
	Metadata map[string]string `json:"metadata"`

	// Properties is a set of key-value pairs, if any, that are associated with the image.
	Properties map[string]string `json:"properties"`

	// CreatedAt is the date when the image has been created.
	CreatedAt time.Time `json:"-"`

	// UpdatedAt is the date when the last change has been made to the image or it's properties.
	UpdatedAt time.Time `json:"-"`

	// File is the trailing path after the glance endpoint that represent the location
	// of the image or the path to retrieve it.
	File string `json:"file"`

	// Schema is the path to the JSON-schema that represent the image or image entity.
	Schema string `json:"schema"`

	// VirtualSize of the image.
	VirtualSize int64 `json:"virtual_size"`

	// Self is the URL for the virtual machine image.
	Self string `json:"self"`

	// DirectURL is the URL to access the image file kept in external store.
	DirectURL string `json:"direct_url"`

	// Locations is a list of objects, each of which describes an image location.
	Locations []string `json:"locations"`
}

func (s *Image) UnmarshalJSON(b []byte) error {
	type tmp Image
	var p *struct {
		tmp
		SizeBytes interface{} `json:"size"`
		CreatedAt string      `json:"created_at"`
		UpdatedAt string      `json:"updated_at"`
	}
	err := json.Unmarshal(b, &p)
	if err != nil {
		return err
	}
	*s = Image(p.tmp)

	switch t := p.SizeBytes.(type) {
	case nil:
		return nil
	case float32:
		s.SizeBytes = int64(t)
	case float64:
		s.SizeBytes = int64(t)
	default:
		return fmt.Errorf("Unknown type for SizeBytes: %v (value: %v)", reflect.TypeOf(t), t)
	}

	s.CreatedAt, err = time.Parse(time.RFC3339, p.CreatedAt)
	if err != nil {
		return err
	}
	s.UpdatedAt, err = time.Parse(time.RFC3339, p.UpdatedAt)

	// TODO: This should be removed once the Image API groups custom properties
	// under a "properties" object.
	err = s.unmarshalCustomProperties(b, p)
	return err
}

// jsonTagKeys gets a list of JSON tag keys defined for the given struct.
func jsonTagKeys(s interface{}) []string {
	t := reflect.TypeOf(s).Elem()
	keys := make([]string, 0, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		var key string
		field := t.Field(i)
		if tagValue, ok := field.Tag.Lookup("json"); ok {
			key = strings.Split(tagValue, ",")[0]
			if key == "-" {
				continue
			}
		}
		if key == "" {
			// not specified so use field name
			key = strings.ToLower(field.Name)
		}
		keys = append(keys, key)
	}
	return keys
}

// unmarshalCustomProperties parses the JSON-encoded custom properties and
// stores the result in the Image.Properties field.
//
// The OpenStack API allows custom key:value properties to be specified
// when creating images.  As of the Newton release, these custom properties
// are not contained within an explicit "properties" JSON object.  Rather,
// they are key:value pairs within the top level JSON response object.
// Therefore, this function is needed to group all the custom properties
// into the Image.Properties field for easy access by clients.
func (s *Image) unmarshalCustomProperties(b []byte, st interface{}) error {
	// Store custom properties that appear as top level JSON key:value pairs.
	custom := make(map[string]interface{})
	err := json.Unmarshal(b, &custom)
	if err != nil {
		return err
	}
	// custom map now holds every key:value pair of the response so filter out
	// all known fields in Image
	fields := jsonTagKeys((*Image)(nil))
	// Filter out additional keys from modified struct
	fields = append(fields, jsonTagKeys(st)...)
	// Remove known fields so only custom properties remain
	for _, field := range fields {
		delete(custom, field)
	}
	// At this point, custom map should only contain custom properties so update
	// the Image.Properties field.
	s.Properties = make(map[string]string)
	for k, v := range custom {
		if value, ok := v.(string); ok {
			s.Properties[k] = value
		}
	}
	return nil
}

type commonResult struct {
	gophercloud.Result
}

// Extract interprets any commonResult as an Image.
func (r commonResult) Extract() (*Image, error) {
	var s *Image
	err := r.ExtractInto(&s)
	return s, err
}

// CreateResult represents the result of a Create operation
type CreateResult struct {
	commonResult
}

// UpdateResult represents the result of an Update operation
type UpdateResult struct {
	commonResult
}

// GetResult represents the result of a Get operation
type GetResult struct {
	commonResult
}

//DeleteResult model
type DeleteResult struct {
	gophercloud.ErrResult
}

// ImagePage represents page
type ImagePage struct {
	pagination.LinkedPageBase
}

// IsEmpty returns true if a page contains no Images results.
func (r ImagePage) IsEmpty() (bool, error) {
	images, err := ExtractImages(r)
	return len(images) == 0, err
}

// NextPageURL uses the response's embedded link reference to navigate to the next page of results.
func (r ImagePage) NextPageURL() (string, error) {
	var s struct {
		Next string `json:"next"`
	}
	err := r.ExtractInto(&s)
	if err != nil {
		return "", err
	}
	return nextPageURL(r.URL.String(), s.Next), nil
}

// ExtractImages interprets the results of a single page from a List() call, producing a slice of Image entities.
func ExtractImages(r pagination.Page) ([]Image, error) {
	var s struct {
		Images []Image `json:"images"`
	}
	err := (r.(ImagePage)).ExtractInto(&s)
	return s.Images, err
}
