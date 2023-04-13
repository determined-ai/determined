import { useObservable } from 'micro-observables';
import { useRef } from 'react';
import { debounce } from 'throttle-debounce';

import Button from 'components/kit/Button';

import css from './FilterForm.module.scss';
import { FilterFormStore, ITEM_LIMIT } from './FilterFormStore';
import FilterGroup from './FilterGroup';
import { FormKind } from './type';

interface Props {
  formStore: FilterFormStore;
}

const FilterForm = ({ formStore }: Props): JSX.Element => {
  const scrollBottomRef = useRef<HTMLDivElement>(null);
  const data = useObservable(formStore.formset);
  const isButtonDisabled = data.filterGroup.children.length > ITEM_LIMIT;

  const onAddItem = (formKind: FormKind) => {
    formStore.addChild(data.filterGroup.id, formKind, data.filterGroup.children.length);
    debounce(100, () => {
      scrollBottomRef?.current?.scrollIntoView({ behavior: 'smooth', block: 'end' });
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
          <Button disabled={isButtonDisabled} type="text" onClick={() => onAddItem(FormKind.Field)}>
            + Add condition
          </Button>
          <Button disabled={isButtonDisabled} type="text" onClick={() => onAddItem(FormKind.Group)}>
            + Add condition group
          </Button>
        </div>
        <Button type="text" onClick={() => formStore.removeChild(data.filterGroup.id)}>
          Clear all
        </Button>
      </div>
      <div style={{ maxWidth: '500px', wordWrap: 'break-word' }}>
        {JSON.stringify(formStore.json)}
      </div>
    </div>
  );
};

export default FilterForm;
