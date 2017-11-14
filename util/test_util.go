package util

import (
	"os"
	"math/rand"
	"os/exec"
	"bytes"
	"strings"
	"time"
	"strconv"
)

var TEST_CONFIG = InitSystemTestConfig()

type SystemTestConfig struct {
	CLIPath          string
	Namespace        string
	KafkaPodName     string
	HTTPGatewayURL   string
	DockerOrg        string
	DockerUsername   string
	DockerPassword   string
	BaseDir          string
	MessageRTTimeout int
}

func InitSystemTestConfig() SystemTestConfig {

	return SystemTestConfig{
		CLIPath:          ensureEnv("SYS_TEST_CLI_PATH"),
		BaseDir:          ensureEnv("SYS_TEST_BASE_DIR"),
		Namespace:        ensureEnv("SYS_TEST_NS"),
		KafkaPodName:     ensureEnv("SYS_TEST_KAFKA_POD_NAME"),
		HTTPGatewayURL:   ensureEnv("SYS_TEST_HTTP_GW_URL"),
		DockerOrg:        ensureEnv("SYS_TEST_DOCKER_ORG"),
		DockerUsername:   ensureEnv("SYS_TEST_DOCKER_USERNAME"),
		DockerPassword:   ensureEnv("SYS_TEST_DOCKER_PASSWORD"),
		MessageRTTimeout: ensureEnvInt("SYS_TEST_MSG_RT_TIMEOUT_SEC"),
	}
}

func ensureEnv(varName string) (string) {
	varValue := os.Getenv(varName)
	if varValue == "" {
		panic("Expected [" + varName + "] environment variable to be set")
	}
	return varValue
}
func ensureEnvInt(varName string) (int) {
	varValue := ensureEnv(varName)
	intValue, err := strconv.Atoi(varValue)
	if err != nil {
		panic("Couldn't parse [" + varName + "/" + varValue + "] as integer")
	}
	return intValue
}

func CLI(directory string, args ...string) {
	runSafely("FaaS CLI", directory, TEST_CONFIG.CLIPath, args...)
}

func SendMessageToGateway(topic string, message string) {
	runSafely("Curl", "/", "curl", "-H'Content-Type:text/plain'", "-d", message, TEST_CONFIG.HTTPGatewayURL+"/messages/"+topic)
}
func KubectlFromKafkaPod(topic string) string {

	outBuffer := bytes.NewBufferString("")
	cmd := exec.Command("kubectl", "-n", TEST_CONFIG.Namespace, "exec", TEST_CONFIG.KafkaPodName, "--", "/opt/kafka/bin/kafka-console-consumer.sh", "--bootstrap-server", "localhost:9092", "--topic", topic, "--from-beginning", "--max-messages", "1")
	cmd.Stdout = outBuffer
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		panic("Kubectl Kafka start failed")
		cmd.Stderr.Write(outBuffer.Bytes())
	}
	timer := time.AfterFunc(time.Duration(TEST_CONFIG.MessageRTTimeout)*time.Second, func() {
		cmd.Process.Kill()
		panic("Kubectl Kafka timed out")
		cmd.Stderr.Write(outBuffer.Bytes())
	})
	err := cmd.Wait()
	timer.Stop()

	if err != nil {
		panic("Kubectl Kafka failed")
	}
	return outBuffer.String()
}

func DockerTagAndPush(functionName string, imageName string) {

	outBuffer := bytes.NewBufferString("")
	dockerCmd := exec.Command("docker", "images", "-f", "reference="+functionName, "--format", "{{.ID}}")
	dockerCmd.Stdout = outBuffer
	dockerCmd.Stderr = os.Stderr
	err := dockerCmd.Run()
	if err != nil {
		panic("Docker image list failed")
		dockerCmd.Stderr.Write(outBuffer.Bytes())
	}

	runSafely("Docker Tag", "/", "docker", "tag", strings.TrimSpace(outBuffer.String()), imageName)
	runSafely("Docker Login", "/", "docker", "login", "-u", TEST_CONFIG.DockerUsername, "-p", TEST_CONFIG.DockerPassword)
	runSafely("Docker Push", "/", "docker", "push", imageName)
}

func MvnCleanPackage(directory string) {
	runSafely("Maven Build", directory, "mvn", "clean", "package")
}

func runSafely(description string, directory string, command string, args ...string) {
	cmd := exec.Command(command, args...)
	cmd.Dir = directory
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		panic(description + " failed in directory " + directory)
	}
}

const letters = "abcdefghijklmnopqrstuvwxyz"

func RandStringShort() string {
	return randString(8)
}

func RandStringLong() string {
	return randString(24)
}

func randString(n int) string {
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}