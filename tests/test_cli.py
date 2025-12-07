import tempfile
from pathlib import Path
import yaml

from uptui import cli


def test_load_config_yaml(tmp_path):
    p = tmp_path / "cfg.yaml"
    content = {"monitors": [{"name": "X", "url": "https://x.example"}]}
    p.write_text(yaml.safe_dump(content), encoding="utf-8")

    data = cli.load_config(p)
    assert isinstance(data, dict)
    assert "monitors" in data
    assert data["monitors"][0]["name"] == "X"
