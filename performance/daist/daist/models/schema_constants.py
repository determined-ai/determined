from typing import NewType

from .base import BaseEnum


class Property(BaseEnum):
    type_ = NewType('Property', str)

    ID: type_ = '$id'
    SCHEMA: type_ = '$schema'

    all_ = (ID, SCHEMA)


class Format(BaseEnum):
    type_ = NewType('Format', str)

    DATE_TIME: type_ = 'date-time'

    all_ = (DATE_TIME, )


class Key(BaseEnum):
    type_ = NewType('Key', str)

    ADDITIONAL_PROPERTIES: type_ = 'additionalProperties'
    ANY_OF: type_ = 'anyOf'
    DEPENDENCIES: type_ = 'dependencies'
    DEPENDENT_REQUIRED: type_ = 'dependentRequired'
    DESCRIPTION: type_ = 'description'
    EXAMPLES: type_ = 'examples'
    EXCLUSIVE_MAXIMUM: type_ = 'exclusiveMaximum'
    EXCLUSIVE_MINIMUM: type_ = 'exclusiveMinimum'
    FORMAT: type_ = 'format'
    ID: type_ = Property.ID
    ITEMS: type_ = 'items'
    MAXIMUM: type_ = 'maximum'
    MAX_LENGTH: type_ = 'maxLength'
    MIN_ITEMS: type_ = 'minItems'
    MINIMUM: type_ = 'minimum'
    MIN_LENGTH: type_ = 'minLength'
    ONE_OF: type_ = 'oneOf'
    PATTERN: type_ = 'pattern'
    PATTERN_PROPERTIES: type_ = 'patternProperties'
    PROPERTIES: type_ = 'properties'
    REQUIRED: type_ = 'required'
    SCHEMA: type_ = Property.SCHEMA
    TITLE: type_ = 'title'
    TYPE: type_ = 'type'
    UNIQUE_ITEMS: type_ = 'uniqueItems'

    all_ = (ADDITIONAL_PROPERTIES, ANY_OF, DEPENDENCIES, DEPENDENT_REQUIRED, DESCRIPTION,
            EXAMPLES, EXCLUSIVE_MAXIMUM, EXCLUSIVE_MINIMUM, FORMAT, ID, ITEMS, MAXIMUM, MAX_LENGTH,
            MIN_ITEMS, MINIMUM, MIN_LENGTH, ONE_OF, PATTERN, PATTERN_PROPERTIES, PROPERTIES,
            REQUIRED, SCHEMA, TITLE, TYPE, UNIQUE_ITEMS)


class Type(BaseEnum):
    type_ = NewType('Type', str)

    ARRAY: type_ = 'array'
    BOOLEAN: type_ = 'boolean'
    INTEGER: type_ = 'integer'
    NULL: type_ = 'null'
    NUMBER: type_ = 'number'
    OBJECT: type_ = 'object'
    STRING: type_ = 'string'

    all_ = (ARRAY, BOOLEAN, INTEGER, NULL, NUMBER, OBJECT, STRING)


class SchemaKw(object):
    """
    This class provides python addressing to jsonschema keywords.
    """
    property = Property
    format = Format
    key = Key
    type = Type
