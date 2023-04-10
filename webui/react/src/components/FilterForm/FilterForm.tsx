import { useObservable } from 'micro-observables';
import { useRef } from 'react';
import { debounce } from 'throttle-debounce';

import Button from 'components/kit/Button';

import css from './FilterForm.module.scss';
import { FilterFormStore } from './FilterFormStore';
import FilterGroup from './FilterGroup';
import { FormType } from './type';

interface Props {
  formStore: FilterFormStore;
}

const FilterForm = ({ formStore }: Props): JSX.Element => {
  const scrollBottomRef = useRef<HTMLDivElement>(null);
  const data = useObservable(formStore.formset);

  const onAddItem = (formType: FormType) => {
    formStore.addChild(data.filterGroup.id, formType, data.filterGroup.children.length);
    debounce(500, () => {
      scrollBottomRef?.current?.scrollIntoView({
        behavior: 'smooth',
        block: 'end',
      });
    })();
  };

  return (
    <div className={css.base}>
      <div className={css.filter}>
        <FilterGroup
          conjunction={data.filterGroup.conjunction}
          formStore={formStore}
          group={data.filterGroup}
          index={0}
          level={0}
          parentId={data.filterGroup.id}
        />
        <div ref={scrollBottomRef} />
      </div>
      <div className={css.buttonContainer}>
        <div className={css.addButtonContainer}>
          <Button type="text" onClick={() => onAddItem(FormType.Field)}>
            + Add condition field
          </Button>
          <Button type="text" onClick={() => onAddItem(FormType.Group)}>
            + Add condition group
          </Button>
        </div>
        <Button type="text" onClick={() => formStore.removeChild(data.filterGroup.id)}>
          Clear All
        </Button>
      </div>
      <div style={{ maxWidth: '500px', wordWrap: 'break-word' }}>{formStore.query}</div>
    </div>
  );
};

export default FilterForm;
