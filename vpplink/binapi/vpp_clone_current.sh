#!/bin/bash
VPP_COMMIT=884058096

if [ ! -d $1 ]; then
	git clone "https://gerrit.fd.io/r/vpp" $1
	cd $1
	git reset --hard ${VPP_COMMIT}
else
	cd $1
	git fetch "https://gerrit.fd.io/r/vpp" && git reset --hard ${VPP_COMMIT}
fi

# TODO git fetch "https://gerrit.fd.io/r/vpp" refs/changes/86/29386/7 && git cherry-pick FETCH_HEAD # 29386: virtio: DRAFT: multi tx support | https://gerrit.fd.io/r/c/vpp/+/29386
git fetch "https://gerrit.fd.io/r/vpp" refs/changes/67/30467/1 && git cherry-pick FETCH_HEAD # 30467: tap: fix the buffering index for gro | https://gerrit.fd.io/r/c/vpp/+/30467
# ------------- interrupt patches -------------
# git fetch "https://gerrit.fd.io/r/vpp" refs/changes/08/29808/22 && git cherry-pick FETCH_HEAD # 29808: interface: rx queue infra rework, part one | https://gerrit.fd.io/r/c/vpp/+/29808
# git fetch "https://gerrit.fd.io/r/vpp" refs/changes/28/30128/1 && git cherry-pick FETCH_HEAD # 30128: virtio: update interrupt mode to new infra | https://gerrit.fd.io/r/c/vpp/+/30128
# Interrupt 10us ?
# ------------- interrupt patches -------------

# ------------- Cnat patches -------------
git fetch "https://gerrit.fd.io/r/vpp" refs/changes/55/29955/4 && git cherry-pick FETCH_HEAD # 29955: cnat: Fix throttle hash & cleanup | https://gerrit.fd.io/r/c/vpp/+/29955
git fetch "https://gerrit.fd.io/r/vpp" refs/changes/73/30273/2 && git cherry-pick FETCH_HEAD # 30273: cnat: Fix session with deleted tr | https://gerrit.fd.io/r/c/vpp/+/30273
git fetch "https://gerrit.fd.io/r/vpp" refs/changes/75/30275/8 && git cherry-pick FETCH_HEAD # 30275: cnat: add input feature node | https://gerrit.fd.io/r/c/vpp/+/30275
git fetch "https://gerrit.fd.io/r/vpp" refs/changes/87/28587/28 && git cherry-pick FETCH_HEAD # 28587: cnat: k8s extensions
# ------------- Cnat patches -------------

# ------------- Policies patches -------------
git fetch "https://gerrit.fd.io/r/vpp" refs/changes/83/28083/14 && git cherry-pick FETCH_HEAD # 28083: acl: acl-plugin custom policies |  https://gerrit.fd.io/r/c/vpp/+/28083
git fetch "https://gerrit.fd.io/r/vpp" refs/changes/13/28513/15 && git cherry-pick FETCH_HEAD # 25813: capo: Calico Policies plugin | https://gerrit.fd.io/r/c/vpp/+/28513
# ------------- Policies patches -------------




