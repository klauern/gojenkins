FROM jenkins/jenkins:alpine
EXPOSE 8080

ENV JAVA_OPTS -Djenkins.install.runSetupWizard=false

RUN /usr/local/bin/install-plugins.sh cloudbees-folder ssh-slaves
