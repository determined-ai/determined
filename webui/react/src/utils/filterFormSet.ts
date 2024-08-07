import {
  Conjunction,
  FormFieldWithoutId,
  FormGroupWithoutId,
} from 'components/FilterForm/components/type';

/**
 * build a new filter group given an existing one and a child. will add the child
 * as a child of the current formGroup if the conjunction matches, otherwise
 * will create a new wrapper group having both as children
 */
export const combine = (
  filterGroup: FormGroupWithoutId,
  conjunction: Conjunction,
  child: FormGroupWithoutId | FormFieldWithoutId,
): FormGroupWithoutId => {
  if (filterGroup.conjunction === conjunction) {
    return {
      ...filterGroup,
      children: [...filterGroup.children, child],
    };
  }
  return {
    children: [filterGroup, child],
    conjunction,
    kind: 'group',
  };
};
