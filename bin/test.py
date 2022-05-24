#!/usr/bin/env python
import subprocess
import pathlib

SAAS_REPO='https://github.com/determined-ai/saas'
CORE_REPO='https://github.com/determined-ai/determined'
SM_DIR = 'src/shared'


# Python program to print
# colored text and background
def print_purple(skk): print("\033[93m {}\033[00m" .format(skk))


def run(command):
    print_purple(f'{command}')
    subprocess.run(command, check=True, shell=True)


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


def build_saas(sm_hash: str):
    clone_dir = '/tmp' / pathlib.Path('saas')
    web_dir = 'web'
    run(f'rm -rf {clone_dir}; git clone {SAAS_REPO} {clone_dir} --recurse-submodules')
    run(f'cd {clone_dir} && git checkout master && git pull')
    run(f'cd {clone_dir}/{web_dir}/{SM_DIR} && git checkout {sm_hash}')
    run(f'cd {clone_dir} && make -C {web_dir} node_modules/done.stamp build')


def build_core(sm_hash: str):
    clone_dir = '/tmp' / pathlib.Path('core')
    web_dir = 'webui/react'
    run(f'rm -rf {clone_dir}; git clone {CORE_REPO} {clone_dir} --recurse-submodules')
    run(f'cd {clone_dir} && git checkout master && git pull')
    run(f'cd {clone_dir}/{web_dir}/{SM_DIR} && git checkout {sm_hash}')
    run(f'cd {clone_dir} && make -C {web_dir} node_modules/done.stamp build')


if __name__ == '__main__':
    build_saas(get_current_hash())
    build_core(get_current_hash())
