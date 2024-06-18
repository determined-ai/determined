import * as t from 'io-ts';

/**
 * Given a type that has props, return the underlying props. This is identical
 * to an internal io-ts function used for the exact combinator.
 */
export const getProps = <T extends t.HasProps | t.ExactC<t.HasProps>>(codec: T): t.Props => {
  switch (codec._tag) {
    case 'RefinementType':
    case 'ReadonlyType':
    case 'ExactType':
      return getProps(codec.type);
    case 'StrictType':
    case 'PartialType':
    case 'InterfaceType':
      return codec.props;
    case 'IntersectionType':
      return codec.types.reduce((acc, type) => ({ ...acc, ...getProps(type) }), {});
  }
};

/**
 * Given an io-ts codec, determine if it has props. Used to guard any type codec
 * to HasProps in order to eventually get the props
 */
export const isHasProps = (codec: t.Mixed): codec is t.HasProps => {
  return (
    codec instanceof t.StrictType ||
    codec instanceof t.PartialType ||
    codec instanceof t.InterfaceType ||
    ((codec instanceof t.RefinementType ||
      codec instanceof t.ReadonlyType ||
      codec instanceof t.ExactType) &&
      isHasProps(codec.type)) ||
    (codec instanceof t.IntersectionType && codec.types.every(isHasProps))
  );
};
