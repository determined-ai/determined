import sys
from determined.cli.cli import main, all_args_description, make_parser

if __name__ == "__main__":
    if len(sys.argv) > 1 and sys.argv[1] == 'deploy':
        from determined.deploy.cli import args_description as deploy_args_description
        all_args_description.append(deploy_args_description)
    main(make_parser(all_args_description))
