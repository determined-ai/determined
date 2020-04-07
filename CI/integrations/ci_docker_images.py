import argparse
import re
import typing


def get_images(images_str: str) -> typing.List[str]:
    if images_str:
        return re.split(r"\s+|,", images_str)

    return []


def get_pull_tag_str(registry: str, images: typing.List[str]) -> str:
    ecr_images = [f"{registry}{im}" for im in images]
    image_pulls = [f"docker pull {ecr_im}" for ecr_im in ecr_images]
    image_tags = [f"docker tag {ecr_im} {im}" for ecr_im, im in zip(ecr_images, images)]
    image_pull_str = "\n".join(image_pulls + image_tags)
    return image_pull_str


def tag_agent_master_version(images: typing.List[str], version: str) -> str:
    agent_image = list(filter(lambda s: "agent" in s, images))[0]
    master_image = list(filter(lambda s: "master" in s, images))[0]
    return "\n".join(
        [
            f"docker tag {agent_image} determinedai/determined-agent:{version}",
            f"docker tag {master_image} determinedai/determined-master:{version}",
        ]
    )


def main() -> None:
    parser = argparse.ArgumentParser(description="Integration test docker image helper.")
    parser.add_argument("--registry", default="", help="The docker registry to pull images from.")
    parser.add_argument("--version", default="", help="Version number for agent and master.")
    parser.add_argument(
        "--environments-only", action="store_true", help="Only download environments."
    )
    parser.add_argument("images", type=str, help="List of docker images to pull.")
    args = parser.parse_args()

    images = get_images(args.images)
    print(get_pull_tag_str(args.registry, images))
    if not args.environments_only:
        print(tag_agent_master_version(images, args.version))


if __name__ == "__main__":
    main()
