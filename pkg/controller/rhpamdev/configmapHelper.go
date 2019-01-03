package rhpamdev

import (
	"fmt"
	"io/ioutil"
	"os"
)

type ConfigmapHelper struct {
	resourcePath string
}

func newConfigmapHelper() *ConfigmapHelper {

	resourcePath := os.Getenv("RESOURCE_PATH")
	if resourcePath == "" {
		resourcePath = "./resources"
	}

	return &ConfigmapHelper{
		resourcePath: resourcePath,
	}
}

func (c *ConfigmapHelper) data(resources []string) (map[string]string, error) {

	data := make(map[string]string)

	for _, resourceName := range resources {
		s, err := c.loadFile(resourceName)
		if err != nil {
			return nil, fmt.Errorf("Error reading resource %s: %s", resourceName, err)
		}
		data[resourceName] = s
	}
	return data, nil
}

func (c *ConfigmapHelper) loadFile(name string) (string, error) {
	path := fmt.Sprintf("%s/%s", c.resourcePath, name)
	res, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}

	return string(res[:]), nil
}
