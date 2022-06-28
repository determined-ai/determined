#!/usr/bin/env python3
import subprocess
import typing as t
import pathlib
import argparse

parser = argparse.ArgumentParser()

SHARED_WEB_REPO='https://github.com/determined-ai/shared-web'
SAAS_REPO='https://github.com/determined-ai/saas'
CORE_REPO='https://github.com/determined-ai/determined'
SHARED_DIR = 'src/shared'

repos = {
    'saas': {
        'repo': SAAS_REPO,
        'web_dir': 'web',
        'using_sm': True,
    },
    'core': {
        'repo': CORE_REPO,
        'web_dir': 'webui/react',
        'using_sm': False,
    },
}


# print colored text
def print_colored(skk): print("\033[93m {}\033[00m" .format(skk))


def run(command, cwd: t.Optional[pathlib.Path] = None):
    print_colored(f'{command} [cwd: {cwd}]')
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
    return result.stdout.decode('utf-8')


# get current git hash
def get_current_hash():
    return get_output('git rev-parse HEAD')


def setup_user(user, name: str, sm_hash: t.Optional[str] = None, repo_hash: t.Optional[str] = None):
    repo_hash = repo_hash or 'master'
    clone_dir = '/tmp' / pathlib.Path(name)
    web_dir = clone_dir / user['web_dir']
    run(f'rm -rf {clone_dir}; git clone {user["repo"]} {clone_dir} --recurse-submodules')
    run(f'git checkout {repo_hash}', cwd=clone_dir)

    if sm_hash is None: return clone_dir
    # update the shared code
    if user['using_sm']:
        run(f'git checkout {sm_hash}', cwd=web_dir/SHARED_DIR)
    else:
        rel_shared_dir = pathlib.Path(user['web_dir'])/SHARED_DIR
        cmd = f'git subtree pull --prefix {rel_shared_dir} {SHARED_WEB_REPO} {sm_hash} --squash -m "update shared code"'
        run(cmd, cwd=clone_dir)
    return clone_dir



def run_webui_tests(web_dir: pathlib.Path):
    """
    runs webui tests. requires web_dir to be set up
    """
    for target in ['get-deps', 'check', 'build', 'test']:
        run(f'make {target}', cwd=web_dir)


def overwrite_with_cur_shared(user, name: str):
    """
    overrides the shared code in a repo with the shared
    directory from the current local clone.
    """
    clone_dir = '/tmp' / pathlib.Path(name)
    web_dir = clone_dir / user['web_dir']
    run(f'rm -rf {clone_dir}; git clone {user["repo"]} {clone_dir} --recurse-submodules')
    dest = web_dir / SHARED_DIR
    src = pathlib.Path.cwd() # FIXME find out where the current shared dir is relative to the test script
    run(f'rm -rf {dest}; mkdir -p {dest}')
    run(f'cp -r {src} {dest}')


def test(sm_hash: str, repo_names: t.List[str]):
    for name in repo_names:
        if name not in repos:
            raise ValueError(f'unknown repo name: {name}')
        user = repos[name]
        clone_dir = setup_user(user, name, sm_hash)
        web_dir = clone_dir / user['web_dir']
        run_webui_tests(web_dir)

def test_with_current_shared(repo_names: t.List[str]):
    """
    test the current shared code with target repos.
    """
    for name in repo_names:
        if name not in repos:
            raise ValueError(f'unknown repo name: {name}')
        user = repos[name]
        clone_dir = setup_user(user, name)
        web_dir = clone_dir / user['web_dir']
        overwrite_with_cur_shared(user, name)
        run_webui_tests(web_dir)


if __name__ == '__main__':
    parser.add_argument('--sw-hash', help='desired shared-web githash to test')
    parser.add_argument('--test-local',
                        help='test the repositories with the current shared code',
                        action='store_true', default=False)
    parser.add_argument('--repos',
                        help=f'repos to test. available: {", ".join(repos.keys())}',
                        nargs='+')
    args = parser.parse_args()
    repos_to_test = args.repos or repos.keys()
    # we cannot deduce the current shared-web git hash when using subtree
    for repo in repos_to_test:
        if not repos[repo]['using_sm'] and args.sw_hash is None:
            raise argparse.ArgumentError(None, '--sw-hash is required for subtree setup')
    # get current git hash from cli args as first argument through sys.args if one is provided
    sm_hash = args.sw_hash if args.sw_hash else get_current_hash()

    if args.test_local:
        if pathlib.Path.cwd().name != 'shared':
            raise argparse.ArgumentError(None,
                                         '--test-local must be run from the shared directory root')
        test_with_current_shared(repos_to_test)
    else:
        test(sm_hash, repos_to_test)
