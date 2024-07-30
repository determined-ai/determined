def parse_class_and_method_name_from_test_id(id_: str):
    return '.'.join(id_.split('.')[-2:])


def to_snake_case(a_str: str) -> str:
    return a_str.lower().replace(' ', '_').replace('-', '_')
