import { useObservable } from 'micro-observables';
import { useRef } from 'react';

import { FilterFormStore, ITEM_LIMIT } from 'components/FilterForm/components/FilterFormStore';
import FilterGroup from 'components/FilterForm/components/FilterGroup';
import { FormKind } from 'components/FilterForm/components/type';
import Button from 'components/kit/Button';
import Spinner from 'components/kit/Spinner';
import Toggle from 'components/kit/Toggle';
import { Loadable } from 'components/kit/utils/loadable';
import { V1ProjectColumn } from 'services/api-ts-sdk';

import css from './FilterForm.module.scss';

interface Props {
  formStore: FilterFormStore;
  columns: V1ProjectColumn[];
  onHidePopOver: () => void;
}

const FilterForm = ({ formStore, columns, onHidePopOver }: Props): JSX.Element => {
  const scrollBottomRef = useRef<HTMLDivElement>(null);
  const loadableData = useObservable(formStore.formset);
  const isButtonDisabled = Loadable.match(loadableData, {
    _: () => true,
    Loaded: (data) => data.filterGroup.children.length > ITEM_LIMIT,
  });

  const onAddItem = (formKind: FormKind) => {
    Loadable.forEach(loadableData, (data) => {
      formStore.addChild(data.filterGroup.id, formKind);
      setTimeout(() => {
        scrollBottomRef?.current?.scrollIntoView({ behavior: 'smooth', block: 'end' });
      }, 100);
    });
  };

  return (
    <div className={css.base}>
      {Loadable.match(loadableData, {
        Failed: () => null,
        Loaded: (data) => (
          <>
            <div className={css.header}>
              <div>Show experiments…</div>
              <Toggle
                checked={data.showArchived}
                label="Show Archived"
                onChange={() => formStore.setArchivedValue(!data.showArchived)}
              />
            </div>
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
                <Button
                  disabled={isButtonDisabled}
                  type="text"
                  onClick={() => onAddItem(FormKind.Field)}>
                  + Add condition
                </Button>
                <Button
                  disabled={isButtonDisabled}
                  type="text"
                  onClick={() => onAddItem(FormKind.Group)}>
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
          </>
        ),
        NotLoaded: () => <Spinner spinning />, // TODO correctly handle error state
      })}
    </div>
  );
};

export default FilterForm;
