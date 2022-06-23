#!/usr/bin/env python3
import subprocess
import typing as t
import pathlib
import argparse

parser = argparse.ArgumentParser()

SAAS_REPO='https://github.com/determined-ai/saas'
CORE_REPO='https://github.com/determined-ai/determined'
SM_DIR = 'src/shared'

repos = {
    'saas': {
        'repo': SAAS_REPO,
        'web_dir': 'web',
    },
    'core': {
        'repo': CORE_REPO,
        'web_dir': 'webui/react',
    },
}


# Python program to print
# colored text and background
def print_purple(skk): print("\033[93m {}\033[00m" .format(skk))


def run(command, cwd: t.Optional[pathlib.Path] = None):
    print_purple(f'{command}')
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


def setup_user(user, name: str, sm_hash: str):
    clone_dir = '/tmp' / pathlib.Path(name)
    web_dir = clone_dir / user['web_dir']
    run(f'rm -rf {clone_dir}; git clone {user["repo"]} {clone_dir} --recurse-submodules')
    run(f'git checkout master && git pull', cwd=clone_dir)
    run(f'git checkout {sm_hash}', cwd=web_dir/SM_DIR)


def test(sm_hash: str, repo_names: t.List[str]):
    for name in repo_names:
        if name not in repos:
            raise ValueError(f'unknown repo name: {name}')
        user = repos[name]
        setup_user(user, name, sm_hash)
        clone_dir = '/tmp' / pathlib.Path(name)
        web_dir = clone_dir / user['web_dir']
        for target in ['get-deps', 'check', 'build', 'test']:
            run(f'make {target}', cwd=web_dir)


if __name__ == '__main__':
    parser.add_argument('--git-hash', help='desired shared-web githash to test')
    parser.add_argument('--repos',
                        help=f'repos to test. available: {", ".join(repos.keys())}',
                        nargs='+')
    args = parser.parse_args()
    # get current git hash from cli args as first argument through sys.args if one is provided
    sm_hash = args.git_hash if args.git_hash else get_current_hash()
    test(sm_hash, args.repos or repos.keys())
