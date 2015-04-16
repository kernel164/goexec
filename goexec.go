package main

import (
	"fmt"
	"gopkg.in/codegangsta/cli.v1"
	"gopkg.in/yaml.v1"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"path/filepath"
)


type Config struct {
	Program            string
	Args               []string
	Envs               []string
	Preprocess_dirs    []string
}

func check(e error) {
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
		//panic(e)
	}
}

func expandValue(text string) string {
	return os.ExpandEnv(text)
}

func main() {
	// defer cleanup()
	app := cli.NewApp()
	app.Name = "goexec"
	app.Usage = "simple exec wrapper"
	app.Version = "0.1"
	app.Author = "nobody"
	app.Usage = "goplay [global options] command"
	app.Flags = []cli.Flag{
		cli.StringFlag{Name: "file, f", Value: "exec.yml", Usage: "exec file"},
		cli.StringFlag{Name: "env-file, e", Value: "env.yml", Usage: "env file"},
		cli.StringSliceFlag{Name: "env, E", Value: &cli.StringSlice{}, Usage: "env variables"},
		cli.StringSliceFlag{Name: "path, P", Value: &cli.StringSlice{}, Usage: "preprocess dir paths"},
	}
	app.Action = func(c *cli.Context) {
		cmd := c.Args().First()
		file := c.String("file")
		efile := c.String("env-file")
		envVars := c.StringSlice("env")
		preprocessPaths := c.StringSlice("path")

		// check config file
		if _, err := os.Stat(file); os.IsNotExist(err) {
			fmt.Printf("Can't find %s. Are you in the right directory?\n", file)
			os.Exit(1)
		}

		if len(cmd) == 0 {
			fmt.Printf("default settings not found?\n")
			os.Exit(1)
		}

		// read the config file.
		filedata, ioerr := ioutil.ReadFile(file)
		check(ioerr)

		parsedCmdConfig := Config{}
		m := make(map[interface{}]interface{})
		yerr := yaml.Unmarshal([]byte(filedata), &m)
		check(yerr)
		cmdConfig, ok := m[cmd]
		if !ok {
			fmt.Printf("Can't find '%s' in %s. Have you defined config for '%s' in the file?\n", cmd, file, cmd)
			os.Exit(1)
		}

		typedCmdConfig := cmdConfig.(map[interface{}]interface{})

		cmdConfigRawData, ymerr := yaml.Marshal(&typedCmdConfig)
		check(ymerr)

		uerr := yaml.Unmarshal(cmdConfigRawData, &parsedCmdConfig)
		check(uerr)

		exec, berr := exec.LookPath(parsedCmdConfig.Program)
		check(berr)

		// check env file (use only if present)
		if _, env_file_err := os.Stat(efile); !os.IsNotExist(env_file_err) {
			envfiledata, envioerr := ioutil.ReadFile(efile)
			check(envioerr)
			envmap := make(map[interface{}]interface{})
			envyerr := yaml.Unmarshal([]byte(envfiledata), &envmap)
			check(envyerr)
			for k, v := range envmap {
				serr := os.Setenv(fmt.Sprintf("%v", k), fmt.Sprintf("%v", v))
				check(serr)
			}
		}

		for _, envVal := range envVars {
			envVals := strings.Split(envVal, "=")
			serr := os.Setenv(envVals[0], envVals[1])
			check(serr)
		}

		for _, envVal := range parsedCmdConfig.Envs {
			envVals := strings.Split(envVal, "=")
			serr := os.Setenv(envVals[0], envVals[1])
			check(serr)
		}

		// run exec
		args := []string{parsedCmdConfig.Program}
		if len(parsedCmdConfig.Args) > 0 {
			args = append(args, parsedCmdConfig.Args...)
		}

		fmt.Printf("%v\n", strings.Join(args, " "))
		preprocess(parsedCmdConfig.Preprocess_dirs);
		preprocess(preprocessPaths);
		env := os.Environ()
		exeerr := syscall.Exec(exec, args, env)
		check(exeerr) // not reachable
	}

	app.Run(os.Args)
}

func visit(path string, f os.FileInfo, err error) error {
	if f.IsDir() {
		return nil
	}
	fmt.Fprintln(os.Stdout, "Processing: " + path)
	blob, ioerr := ioutil.ReadFile(path)
	check(ioerr)
	ioutil.WriteFile(path, []byte(expandValue(string(blob))), f.Mode())
	return nil
}

func preprocess(paths []string) error {
	if len(paths) > 0 {
		for i := 0; i < len(paths); i++ {
			path := paths[i]
			fileinfo, _ := os.Stat(path)
			if fileinfo.IsDir() {
				err := filepath.Walk(path, visit)
				check(err)
			} else {
				fmt.Fprintln(os.Stdout, "Processing " + path)
				visit(path, fileinfo, nil)
			}
		}
	}
	return nil
}
