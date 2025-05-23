# :seedling: Kubean

> [简体中文](./README_zh.md)

<div align="center">

  <p>

[<img src="docs/overrides/assets/images/certified_k8s.png" height=120>](https://github.com/cncf/k8s-conformance/pull/2240)
[<img src="docs/overrides/assets/images/kubean_logo.png" height=120>](https://kubean-io.github.io/website/)
<!--
Source: https://github.com/cncf/artwork/tree/master/projects/kubernetes/certified-kubernetes
-->
  </p>

  <p>

Kubean is a production-ready Kubernetes cluster lifecycle management toolchain, based on [kubespray](https://github.com/kubernetes-sigs/kubespray).

  </p>

[![main workflow](https://github.com/kubean-io/kubean/actions/workflows/auto-main-ci.yaml/badge.svg)](https://github.com/kubean-io/kubean/actions/workflows/auto-main-ci.yaml)
[![codecov](https://codecov.io/gh/kubean-io/kubean/branch/main/graph/badge.svg?token=8FX807D3QQ)](https://codecov.io/gh/kubean-io/kubean)
[![CII Best Practices](https://bestpractices.coreinfrastructure.org/projects/6263/badge)](https://bestpractices.coreinfrastructure.org/projects/6263)
[![kubean coverage](https://raw.githubusercontent.com/dasu23/e2ecoverage/master/badges/kubean/kubeanCoverage.svg)](https://github.com/kubean-io/kubean/blob/main/docs/overrides/test/kubean_testcase.md)
[![license](https://img.shields.io/badge/license-AL%202.0-blue)](https://github.com/kubean-io/kubean/blob/main/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/kubean-io/kubean)](https://goreportcard.com/report/github.com/kubean-io/kubean)
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fkubean-io%2Fkubean.svg?type=shield)](https://app.fossa.com/projects/git%2Bgithub.com%2Fkubean-io%2Fkubean?ref=badge_shield)

</div>

---

<div align="center">
<img src="https://github.com/cncf/artwork/blob/main/other/illustrations/ashley-mcnamara/transparent/cncf-cloud-gophers-transparent.png" style="width:700px;" />
</div>

<div align="center">
**Kubean is a [Cloud Native Computing Foundation sandbox project](https://cncf.io/).**
</div>

## :anchor: Awesome features

- ✨ **Simplicity:** Easily deploy Kubean and manage Kubernetes cluster lifecycles with a powerful declarative API.
- 📦 **Offline Support:** Includes all OS packages, images, and binaries for offline deployment, eliminating resource gathering concerns.
- 🌐 **Compatibility:** Supports multi-arch delivery, including AMD and ARM on common Linux distributions, plus Kunpeng with Kylin.
- 🧩 **Expandability:** Add custom actions to clusters without modifying core Kubespray components.

## :surfing_man: Quick start

### :school: Killercoda tutorials

Try our interactive [Killercoda scenario](https://killercoda.com/kubean) for a hands-on learning experience.

### :computer: Local install

1. Requires an active Kubernetes cluster with Helm installed.

2. Deploy Kubean Operator:

   ```shell
   helm repo add kubean-io https://kubean-io.github.io/kubean-helm-chart/
   helm install kubean kubean-io/kubean --create-namespace -n kubean-system
   ```

   Check the operator status:

   ```shell
   kubectl get pods -n kubean-system
   ```

3. Online deploy an All-In-One cluster with minimal configuration:

   1. Use [AllInOne.yml](./examples/install/1.minimal/), replacing placeholders like `<IP1>` and `<USERNAME>` with your values.

   2. Start `kubeanClusterOps` to run the Kubespray job.

      ```shell
      kubectl apply -f examples/install/1.minimal
      ```

   3. Check the kubespray job status.

      ```shell
      kubectl get job -n kubean-system
      ```

<div align="center">
<a href="https://asciinema.org/a/jFTUi2IdU5yydv88kHkPYMni0"><img src="docs/overrides/assets/images/quick_start.gif" alt="quick_start_image"></a>
</div>

## :ocean: Kubernetes compatibility

| Kubernetes Version | Kubean v0.7.4 | Kubean v0.6.6 | Kubean v0.5.4 | Kubean v0.4.5 | Kubean v0.4.4 |
|:------------------:|:-------------:|:-------------:|:-------------:|:-------------:|:-------------:|
| Kubernetes 1.27    |       ✓       |       ✓       |       ✓       |       ✓       |       ✓       |
| Kubernetes 1.26    |       ✓       |       ✓       |       ✓       |       ✓       |       ✓       |
| Kubernetes 1.25    |       ✓       |       ✓       |       ✓       |       ✓       |       ✓       |
| Kubernetes 1.24    |       ✓       |       ✓       |       ✓       |       ✓       |       ✓       |
| Kubernetes 1.23    |       ✓       |       ✓       |       ✓       |       ✓       |       ✓       |
| Kubernetes 1.22    |       ✓       |       ✓       |       ✓       |       ✓       |       ✓       |
| Kubernetes 1.21    |       ✓       |       ✓       |       ✓       |       ✓       |       ✓       |
| Kubernetes 1.20    |       ✓       |       ✓       |       ✓       |       ✓       |       ✓       |

For a detailed list of Kubernetes versions supported by Kubean, see the [Kubernetes versions list](./docs/zh/usage/support_k8s_version.md).

## :book: Roadmap

View all planned features in our [roadmap](docs/en/develop/roadmap.md).

## :book: Documents

Visit our documentation website: [kubean-io.github.io/kubean/](https://kubean-io.github.io/kubean/)

## :envelope: Join us

Connect with us on the following channels:

- Slack: Join the [#Kubean](https://cloud-native.slack.com/messages/kubean) channel on CNCF Slack (request an [invitation](https://slack.cncf.io/) if needed).
- Email: Find maintainer contacts in [MAINTAINERS.md](./MAINTAINERS.md) for reporting issues or asking questions.

## :thumbsup: Contributors

<div align="center">
<a href="https://github.com/kubean-io/kubean/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=kubean-io/kubean" />
</a>
</div>

## :mag_right: Others

<div align="center">

Copyright The Kubean Authors

We are a [Cloud Native Computing Foundation sandbox project](https://www.cncf.io/).

The Linux Foundation® (TLF) has registered trademarks and uses trademarks. For a list of TLF trademarks, see [Trademark Usage](https://www.linuxfoundation.org/legal/trademark-usage).

</div>

---

<div align="center">
<p>
<img src="https://landscape.cncf.io/images/cncf-landscape-horizontal-color.svg" width="300"/>
<br/><br/>
Kubean enriches the <a href="https://landscape.cncf.io/?selected=kubean">CNCF CLOUD NATIVE Landscape.</a>
</p>
</div>
