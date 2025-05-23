# :seedling: Kubean

> [简体中文](./README_zh.md)

<div align="center">
  <p>
    <!-- 项目认证徽章 -->
    <a href="https://github.com/cncf/k8s-conformance/pull/2240"><img src="docs/overrides/assets/images/certified_k8s.png" height=120></a>
    <a href="https://kubean-io.github.io/website/"><img src="docs/overrides/assets/images/kubean_logo.png" height=120></a>
  </p>

  <p>
    Kubean is a production-ready cluster lifecycle management toolchain based on <a href="https://github.com/kubernetes-sigs/kubespray">kubespray</a> and other cluster LCM engine.
  </p>
  
  <!-- 状态徽章 -->
  <p>
    <a href="https://github.com/kubean-io/kubean/actions/workflows/auto-main-ci.yaml"><img src="https://github.com/kubean-io/kubean/actions/workflows/auto-main-ci.yaml/badge.svg" alt="main workflow"></a>
    <a href="https://codecov.io/gh/kubean-io/kubean"><img src="https://codecov.io/gh/kubean-io/kubean/branch/main/graph/badge.svg?token=8FX807D3QQ" alt="codecov"></a>
    <a href="https://bestpractices.coreinfrastructure.org/projects/6263"><img src="https://bestpractices.coreinfrastructure.org/projects/6263/badge" alt="CII Best Practices"></a>
  </p>
  
  <!-- 测试覆盖率和代码质量徽章 -->
  <p>
    <a href="https://github.com/kubean-io/kubean/blob/main/docs/overrides/test/kubean_testcase.md"><img src="https://raw.githubusercontent.com/dasu23/e2ecoverage/master/badges/kubean/kubeanCoverage.svg" alt="kubean coverage"></a>
    <a href="https://github.com/kubean-io/kubean/blob/main/docs/overrides/test/kubean_testcase.md"><img src="https://raw.githubusercontent.com/dasu23/e2ecoverage/master/badges/kubean/kubeanCoverage2.svg" alt="kubean coverage"></a>
    <a href="https://github.com/kubean-io/kubean/blob/main/LICENSE"><img src="https://img.shields.io/badge/license-AL%202.0-blue" alt="license"></a>
    <a href="https://goreportcard.com/report/github.com/kubean-io/kubean"><img src="https://goreportcard.com/badge/github.com/kubean-io/kubean" alt="Go Report Card"></a>
  </p>
  
  <!-- FOSSA 徽章 -->
  <p>
    <a href="https://app.fossa.com/projects/git%2Bgithub.com%2Fkubean-io%2Fkubean?ref=badge_shield"><img src="https://app.fossa.com/api/projects/git%2Bgithub.com%2Fkubean-io%2Fkubean.svg?type=shield" alt="FOSSA Status"></a>
    <a href="https://app.fossa.com/projects/git%2Bgithub.com%2Fkubean-io%2Fkubean?ref=badge_large"><img src="https://app.fossa.com/api/projects/git%2Bgithub.com%2Fkubean-io%2Fkubean.svg?type=small" alt="FOSSA Status"></a>
  </p>
</div>

---

<p>
<img src="https://github.com/cncf/artwork/blob/main/other/illustrations/ashley-mcnamara/transparent/cncf-cloud-gophers-transparent.png" style="width:700px;" />
</p>

**Kubean is a [Cloud Native Computing Foundation sandbox project](https://cncf.io/).**

## :anchor: Awesome features

- **Simplicity:** Deploying of Kubean and powerful lifecycle management of kubernetes cluster implementing by declarative API.
- **Offline Supported**: Offline packages(os-pkgs, images, binarys) are released with the release. You won't have to worry about how to gather all the resources you need.
- **Compatibility**: Multi-arch delivery Supporting. Such as AMD, ARM with common Linux distributions. Also include Kunpeng with Kylin.
- **Expandability**: Allowing custom actions be added to cluster without any changes for Kubespray.

## :surfing_man: Quick start

### Killercoda tutorials

We created a [scenario](https://killercoda.com/kubean) on [killercoda](https://killercoda.com), which is an online platform for interactive technique learning. You can try it in there.

### Local install

1. Ensure that you have a Kubernetes cluster running, on which Helm is installed

2. Deploy kubean-operator

   ```shell
   helm repo add kubean-io https://kubean-io.github.io/kubean-helm-chart/
   helm install kubean kubean-io/kubean --create-namespace -n kubean-system
   ```

   Then check kubean-operator status by running:

   ```shell
   kubectl get pods -n kubean-system
   ```

3. Online deploy an all-in-one cluster with minimal configuration

   1. A simple way is to use [AllInOne.yml](./examples/install/1.minimal/),
      replacing `<IP1>`, `<USERNAME>`, and other strings with actual values.

   2. Start `kubeanClusterOps` to run the kubespray job.

      ```shell
      kubectl apply -f examples/install/1.minimal
      ```

   3. Check the kubespray job status.

      ```shell
      kubectl get job -n kubean-system
      ```

[![quick_start_image](docs/overrides/assets/images/quick_start.gif)](https://asciinema.org/a/jFTUi2IdU5yydv88kHkPYMni0)

## :ocean: Kubernetes compatibility

|               | Kubernetes 1.27 | Kubernetes 1.26 | Kubernetes 1.25 | Kubernetes 1.24 | Kubernetes 1.23 | Kubernetes 1.22 | Kubernetes 1.21 | Kubernetes 1.20 |
|---------------|:---------------:|:---------------:|:---------------:|:---------------:|:---------------:|:---------------:|:---------------:|:---------------:|
| Kubean v0.7.4 |        ✓        |        ✓        |        ✓        |        ✓        |        ✓        |        ✓        |        ✓        |        ✓        |
| Kubean v0.6.6 |        ✓        |        ✓        |        ✓        |        ✓        |        ✓        |        ✓        |        ✓        |        ✓        |
| Kubean v0.5.4 |        ✓        |        ✓        |        ✓        |        ✓        |        ✓        |        ✓        |        ✓        |        ✓        |
| Kubean v0.4.5 |        ✓        |        ✓        |        ✓        |        ✓        |        ✓        |        ✓        |        ✓        |        ✓        |
| Kubean v0.4.4 |        ✓        |        ✓        |        ✓        |        ✓        |        ✓        |        ✓        |        ✓        |        ✓        |

To check the list of Kubernetes versions supported by Kubean, refer to the [Kubernetes versions list](./docs/zh/usage/support_k8s_version.md).

## :book: Roadmap

For detailed information about all the planned features, refer to the [roadmap](docs/en/develop/roadmap.md).

## :book: Documents

Please visit our website: [kubean-io.github.io/kubean/](https://kubean-io.github.io/kubean/)

## :envelope: Join us

You can connect with us on the following channels:

- Slack: join the [#Kubean](https://cloud-native.slack.com/messages/kubean) channel on CNCF Slack by requesting an [invitation](https://slack.cncf.io/) from CNCF Slack. Once you have access to CNCF Slack, you can join the Kubean channel.
- Email: refer to the [MAINTAINERS.md](./MAINTAINERS.md) to find the email addresses of all maintainers. Feel free to contact them via email to report any issues or ask questions.

## :thumbsup: Contributors

<a href="https://github.com/kubean-io/kubean/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=kubean-io/kubean" />
</a>

## :mag_right: Others

Copyright The Kubean Authors

We are a [Cloud Native Computing Foundation sandbox project](https://www.cncf.io/).

The Linux Foundation® (TLF) has registered trademarks and uses trademarks. For a list of TLF trademarks, see [Trademark Usage](https://www.linuxfoundation.org/legal/trademark-usage).

---

<div align="center">
<p>
<img src="https://landscape.cncf.io/images/cncf-landscape-horizontal-color.svg" width="300"/>
<br/><br/>
Kubean enriches the <a href="https://landscape.cncf.io/?selected=kubean">CNCF CLOUD NATIVE Landscape.</a>
</p>
</div>
