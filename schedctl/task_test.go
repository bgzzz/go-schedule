// +build unit

package main

import (
	"io/ioutil"
	"testing"
)

func TestYamlFileErr(t *testing.T) {
	//create file

	data := []byte(testWrongYaml)
	err := ioutil.WriteFile("./TestScheduleTasksErrorWrongFileStruct.yaml", data, 0644)
	if err != nil {
		t.Error(err.Error())
		// or we can skip it
	}

	_, err = ParseTasksFile("./TestScheduleTasksErrorWrongFileStruct.yaml")
	if err == nil {
		t.Errorf("There should be an error on such file %s", string(data))
	}

}

const testOkYaml = `# example of possible task list

# tasks array 
tasks:
  # task contains cmd descriptor
  - cmd: sleep
  # and array of parameters
  - params: 
      - "30"
  - cmd: sleep
    params: 
      - "30"`

func TestYamlFileOk(t *testing.T) {
	//create file

	data := []byte(testOkYaml)
	err := ioutil.WriteFile("./TestScheduleTasksErrorWrongFileStruct.yaml", data, 0644)
	if err != nil {
		t.Error(err.Error())
		// or we can skip it
	}

	tasks, err := ParseTasksFile("./TestScheduleTasksErrorWrongFileStruct.yaml")
	if err != nil {
		t.Errorf("There should not be an error on such file %s", string(data))
	}

	t.Logf("%+v", tasks)

}
