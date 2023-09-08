import os
import webbrowser
from urllib import parse


def parse_master_address(master_address: str) -> parse.ParseResult:
    if master_address.startswith("https://"):
        default_port = 443
    elif master_address.startswith("http://"):
        default_port = 80
    else:
        default_port = 8080
        master_address = "http://{}".format(master_address)
    parsed = parse.urlparse(master_address)
    if not parsed.port:
        parsed = parsed._replace(netloc="{}:{}".format(parsed.netloc, default_port))
    return parsed


def make_url(master_address: str, suffix: str) -> str:
    """@deprecated use make_url_new instead"""
    parsed = parse_master_address(master_address)
    return parse.urljoin(parsed.geturl(), suffix)


def make_url_new(master_address: str, suffix: str) -> str:
    parsed_suffix = parse.urlparse(suffix)
    if parsed_suffix.scheme and parsed_suffix.netloc:
        return make_url(master_address, suffix)
    parsed = parse_master_address(master_address)
    master_url = parsed.geturl().rstrip("/")
    suffix = suffix.lstrip("/")
    separator = "/" if suffix or master_address.endswith("/") else ""
    return "{}{}{}".format(master_url, separator, suffix)


def maybe_upgrade_ws_scheme(master_address: str) -> str:
    parsed = parse.urlparse(master_address)
    if parsed.scheme == "https":
        return parsed._replace(scheme="wss").geturl()
    elif parsed.scheme == "http":
        return parsed._replace(scheme="ws").geturl()
    else:
        return master_address


def make_interactive_task_url(
    task_id: str,
    service_address: str,
    description: str,
    resource_pool: str,
    task_type: str,
    currentSlotsExceeded: bool,
) -> str:
    wait_path = (
        "/jupyter-lab/{}/events".format(task_id)
        if task_type == "jupyter-lab"
        else "/tensorboard/{}/events?tail=1".format(task_id)
    )
    wait_path_url = service_address + wait_path
    public_url = os.environ.get("PUBLIC_URL", "/det")
    wait_page_url = "{}/wait/{}/{}?eventUrl={}&serviceAddr={}".format(
        public_url, task_type, task_id, wait_path_url, service_address
    )
    task_web_url = "{}/interactive/{}/{}/{}/{}/{}?{}".format(
        public_url,
        task_id,
        task_type,
        parse.quote(description),
        resource_pool,
        parse.quote_plus(wait_page_url),
        f"currentSlotsExceeded={str(currentSlotsExceeded).lower()}",
    )
    return task_web_url


def browser_open(host: str, path: str) -> str:
    url = make_url(host, path)
    webbrowser.open(url)
    return url
