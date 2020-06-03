from determined.experimental import Determined

checkpoint = Determined().get_trial(1).top_checkpoint()

checkpoint.add_metadata({"something": {"override": "value"}})

print("ADD")
__import__("pprint").pprint(checkpoint.metadata)

checkpoint.remove_metadata(["test", "something"])
print("REMOVE")
__import__("pprint").pprint(checkpoint.metadata)
