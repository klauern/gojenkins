# Run a local instance

Write-Output "Build the klauern/gojenkins-testing image"
docker build . --tag=klauern/gojenkins-testing

Write-Output "Stop the 'gojenkins-testing' container"
docker stop jenkins

Write-Output "Remove the 'gojenkins-testing' container"
docker rm jenkins

Write-Output "Start a new 'gojenkins-testing' container"
docker run --name jenkins -d -p 8080:8080 klauern/gojenkins-testing
