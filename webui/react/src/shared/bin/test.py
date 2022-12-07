#!/usr/bin/env python3
import subprocess
import typing as t
import pathlib
import argparse

parser = argparse.ArgumentParser()

SHARED_WEB_REPO = "https://github.com/determined-ai/shared-web.git"
SAAS_REPO = "git@github.com:determined-ai/saas.git"
CORE_REPO = "https://github.com/determined-ai/determined.git"
SHARED_DIR = "src/shared"

repos = {
    "saas": {
        "repo": SAAS_REPO,
        "web_dir": "web",
        "using_sm": True,
    },
    "core": {
        "repo": CORE_REPO,
        "web_dir": "webui/react",
        "using_sm": False,
    },
}


# print colored text
def print_colored(skk):
    print("\033[93m {}\033[00m".format(skk))


def run(command, cwd: t.Optional[pathlib.Path] = None):
    msg = command
    if cwd is not None:
        msg = f"{command} [cwd: {cwd}]"
    print_colored(msg)
    subprocess.run(command, cwd=cwd, check=True, shell=True)


def has_output(command):
    result = subprocess.run(command, shell=True, stdout=subprocess.PIPE)
    return len(result.stdout) > 0


def fails(command):
    try:
        subprocess.check_output(command, shell=True, stderr=subprocess.STDOUT)
    except subprocess.CalledProcessError:
        return True
    return False


# read command output
def get_output(command):
    result = subprocess.run(command, shell=True, stdout=subprocess.PIPE)
    return result.stdout.decode("utf-8")


# get current git hash
def get_current_hash():
    return get_output("git rev-parse HEAD")


def setup_user(user, name: str, sm_hash: t.Optional[str] = None, repo_hash: t.Optional[str] = None):
    repo_hash = repo_hash or "master"
    clone_dir = "/tmp" / pathlib.Path(name)
    web_dir = clone_dir / user["web_dir"]
    run(f'rm -rf {clone_dir}; git clone {user["repo"]} {clone_dir} --recurse-submodules')
    run(f"git checkout {repo_hash}", cwd=clone_dir)
    run("make get-deps", cwd=web_dir)  # calls submodule update

    if sm_hash is None:
        return clone_dir
    # update the shared code
    if user["using_sm"]:
        run(f"git checkout {sm_hash}", cwd=web_dir / SHARED_DIR)
    else:
        rel_shared_dir = pathlib.Path(user["web_dir"]) / SHARED_DIR
        cmd = f'git subtree pull --prefix {rel_shared_dir} {SHARED_WEB_REPO} {sm_hash} --squash -m "update shared code"'
        run(cmd, cwd=clone_dir)
    return clone_dir


def run_webui_tests(web_dir: pathlib.Path):
    """
    runs webui tests. requires web_dir to be set up
    """
    for target in ["check", "build", "test"]:
        run(f"make {target}", cwd=web_dir)


def overwrite_with_cur_shared(user, name: str):
    """
    overrides the shared code in a repo with the shared
    directory from the current local clone.
    """
    clone_dir = "/tmp" / pathlib.Path(name)
    web_dir = clone_dir / user["web_dir"]
    dest = web_dir / SHARED_DIR
    run(f"rm -rf {dest}")
    run(f"cp -r . {dest}")  # copy the current dir (SHARED_DIR) to dest


def test(sm_hash: str, repo_names: t.List[str], args):
    for name in repo_names:
        if name not in repos:
            raise ValueError(f"unknown repo name: {name}")
        user = repos[name]
        clone_dir = setup_user(user, name, sm_hash, repo_hash=args.repo_hash)
        web_dir = clone_dir / user["web_dir"]
        run_webui_tests(web_dir)


def test_local_shared(repo_names: t.List[str], args):
    """
    test the current shared code with target repos.
    """
    for name in repo_names:
        if name not in repos:
            raise ValueError(f"unknown repo name: {name}")
        user = repos[name]
        clone_dir = setup_user(user, name, repo_hash=args.repo_hash)
        web_dir = clone_dir / user["web_dir"]
        overwrite_with_cur_shared(user, name)
        run_webui_tests(web_dir)


def get_user_inputs():
    """
    parse and validate user inputs
    """
    parser.add_argument("--sw-hash", help="desired shared-web githash to test with")
    parser.add_argument(
        "--test-local",
        help="test the repositories with the current shared code"
        + "This must be run from the shared-web root",
        action="store_true",
        default=False,
    )
    parser.add_argument(
        "--repos", help=f'repos to test. available: {", ".join(repos.keys())}', nargs="+"
    )
    parser.add_argument(
        "--repo-hash", type=str, help="git hash of the repo to test. defaults to master"
    )
    args = parser.parse_args()
    repos_to_test = args.repos or repos.keys()

    # we cannot deduce the current shared-web git hash when using subtree
    if not args.test_local:
        for repo in repos_to_test:
            if not repos[repo]["using_sm"] and args.sw_hash is None:
                raise argparse.ArgumentError(None, "--sw-hash is required for subtree setup")

    if args.repo_hash and len(repos_to_test) > 1:
        # implementation limitation
        raise argparse.ArgumentError(None, "--repo-hash can only be used with one repo")

    # get current git hash from cli args as first argument through sys.args if one is provided
    sw_hash = args.sw_hash if args.sw_hash else get_current_hash()

    if args.test_local:
        if args.sw_hash:
            raise argparse.ArgumentError(None, "--sw-hash cannot be used with --test-local")
        # best effort check to make sure we're running in the right directory
        # check these files and directories exist in the current directory
        to_check = ["types.ts", "bin", "Makefile", "components", "configs"]
        for f in to_check:
            if not pathlib.Path(f).exists():
                raise argparse.ArgumentError(
                    None,
                    "it looks like this is not running from "
                    + f"shared-web root: {f} does not exist",
                )

    return args, repos_to_test, sw_hash


if __name__ == "__main__":
    args, repos_to_test, sw_hash = get_user_inputs()

    if args.test_local:
        test_local_shared(repos_to_test, args)
    else:
        test(sw_hash, repos_to_test, args)
