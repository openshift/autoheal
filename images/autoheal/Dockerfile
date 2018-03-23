FROM openshift/origin-base

RUN \
INSTALL_PKGS="openshift-autoheal" && \
yum --enablerepo=origin-local-release install -y ${INSTALL_PKGS} && \
rpm --verify ${INSTALL_PKGS} && \
yum clean all

LABEL \
io.k8s.display-name="OpenShift Autoheal" \
io.k8s.description="OpenShift Autoheal" \
io.openshift.tags="openshift"

RUN useradd --no-create-home autoheal
USER autoheal
EXPOSE 9099

CMD [ "/usr/bin/autoheal" ]
