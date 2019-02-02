package keycloak

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"text/template"
)

func templatePath() string {
	templatePath := os.Getenv("TEMPLATE_PATH")
	if templatePath == "" {
		templatePath = "./templates/rhsso"
	}
	return templatePath
}

func loadTemplate(name string, templatePath string, data interface{}) ([]byte, error) {
	path := fmt.Sprintf("%s/%s.json", templatePath, name)
	tpl, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	parsed, err := template.New("realm").Parse(string(tpl))
	if err != nil {
		return nil, err
	}
	var buffer bytes.Buffer
	err = parsed.Execute(&buffer, data)
	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}
