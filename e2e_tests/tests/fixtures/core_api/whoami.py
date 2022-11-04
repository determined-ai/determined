import subprocess


def main():
    id_output = subprocess.check_output(["id"]).decode("utf-8")
    print(f"id output: {id_output}")


if __name__ == "__main__":
    main()
