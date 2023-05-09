import { Switch } from 'antd';
import { useObservable } from 'micro-observables';
import { useId, useRef } from 'react';
import { debounce } from 'throttle-debounce';

import { FilterFormStore, ITEM_LIMIT } from 'components/FilterForm/components/FilterFormStore';
import FilterGroup from 'components/FilterForm/components/FilterGroup';
import { FormKind } from 'components/FilterForm/components/type';
import Button from 'components/kit/Button';
import { V1ProjectColumn } from 'services/api-ts-sdk';

import css from './FilterForm.module.scss';

interface Props {
  formStore: FilterFormStore;
  columns: V1ProjectColumn[];
  onHidePopOver: () => void;
}

const FilterForm = ({ formStore, columns, onHidePopOver }: Props): JSX.Element => {
  const scrollBottomRef = useRef<HTMLDivElement>(null);
  const data = useObservable(formStore.formset);
  const isButtonDisabled = data.filterGroup.children.length > ITEM_LIMIT;
  const showArchivedSwitchId = useId();

  const onAddItem = (formKind: FormKind) => {
    formStore.addChild(data.filterGroup.id, formKind);
    debounce(100, () => {
      scrollBottomRef?.current?.scrollIntoView({ behavior: 'smooth', block: 'end' });
    })();
  };

  return (
    <div className={css.base}>
      <div className={css.showExpText}>Show experiments…</div>
      <div className={css.filter}>
        <FilterGroup
          columns={columns}
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
        <Button
          type="text"
          onClick={() => {
            formStore.removeChild(data.filterGroup.id);
            onHidePopOver();
          }}>
          Reset
        </Button>
      </div>
      <div className={css.showArchived}>
        <Switch
          checked={data.showArchived}
          id={showArchivedSwitchId}
          size="small"
          onChange={() => formStore.setArchivedValue(!data.showArchived)}
        />
        <label htmlFor={showArchivedSwitchId}>Show Archived</label>
      </div>
    </div>
  );
};

export default FilterForm;
