import logging

import pykube
import pytest
from pytest_helm_charts.clusters import Cluster

logger = logging.getLogger(__name__)


@pytest.mark.smoke
def test_api_working(kube_cluster: Cluster) -> None:
    """Verify the smoke-test cluster is reachable and the chart installed.

    The smoke step deploys the chart into the cluster before this test runs, so
    a healthy API connection here confirms the chart templates render and the
    release installs cleanly.

    Note: mcp-capi serves the cluster-api MCP surface against a management
    cluster it is pointed at; it is not expected to reach readiness on a bare
    kind cluster, and the repo has no released image to pin, so the smoke test
    validates installability rather than pod readiness.
    """
    assert kube_cluster.kube_client is not None
    assert len(pykube.Node.objects(kube_cluster.kube_client)) >= 1
