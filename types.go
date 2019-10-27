package cheops

type DockerCredentialsProvider interface {
	GetCredentials() (string, error)
}
