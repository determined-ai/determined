import Button from 'hew/Button';
import Spinner from 'hew/Spinner';
import Toggle from 'hew/Toggle';
import { Loadable } from 'hew/utils/loadable';
import { useObservable } from 'micro-observables';
import { useRef } from 'react';

import { FilterFormStore, ITEM_LIMIT } from 'components/FilterForm/components/FilterFormStore';
import FilterGroup from 'components/FilterForm/components/FilterGroup';
import { FormKind } from 'components/FilterForm/components/type';
import { V1ProjectColumn } from 'services/api-ts-sdk';

import css from './FilterForm.module.scss';

interface Props {
  formStore: FilterFormStore;
  columns: V1ProjectColumn[];
  projectId?: number;
  entityCopy?: string;
  onHidePopOver: () => void;
}

const FilterForm = ({
  formStore,
  columns,
  projectId,
  entityCopy,
  onHidePopOver,
}: Props): JSX.Element => {
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
    <div className={css.base} data-test-component="FilterForm">
      {Loadable.match(loadableData, {
        Failed: () => null, // TODO inform user if data fails to load
        Loaded: (data) => (
          <>
            <div className={css.header} data-test="header">
              <div>{entityCopy ?? 'Show experiments…'}</div>
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
                projectId={projectId}
              />
              <div ref={scrollBottomRef} />
            </div>
            <div className={css.buttonContainer}>
              <div className={css.addButtonContainer}>
                <Button
                  data-test="addCondition"
                  disabled={isButtonDisabled}
                  type="text"
                  onClick={() => onAddItem(FormKind.Field)}>
                  + Add condition
                </Button>
                <Button
                  data-test="addConditionGroup"
                  disabled={isButtonDisabled}
                  type="text"
                  onClick={() => onAddItem(FormKind.Group)}>
                  + Add condition group
                </Button>
              </div>
              <Button
                data-test="clearFilters"
                type="text"
                onClick={() => {
                  formStore.removeChild(data.filterGroup.id);
                  onHidePopOver();
                }}>
                Clear filters
              </Button>
            </div>
          </>
        ),
        NotLoaded: () => <Spinner spinning />,
      })}
    </div>
  );
};

export default FilterForm;
