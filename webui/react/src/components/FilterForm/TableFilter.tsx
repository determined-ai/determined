import Button from 'hew/Button';
import Dropdown from 'hew/Dropdown';
import Icon from 'hew/Icon';
import { Loadable } from 'hew/utils/loadable';
import { useObservable } from 'micro-observables';
import { useCallback } from 'react';

import FilterForm from 'components/FilterForm/components/FilterForm';
import { FilterFormStore } from 'components/FilterForm/components/FilterFormStore';
import { FormKind } from 'components/FilterForm/components/type';
import { V1ProjectColumn } from 'services/api-ts-sdk';

interface Props {
  loadableColumns: Loadable<V1ProjectColumn[]>;
  bannedFilterColumns?: Set<string>;
  formStore: FilterFormStore;
  isMobile?: boolean;
  isOpenFilter: boolean;
  projectId?: number;
  entityCopy?: string;
  onIsOpenFilterChange?: (value: boolean) => void;
}

const TableFilter = ({
  loadableColumns,
  bannedFilterColumns,
  formStore,
  isMobile = false,
  isOpenFilter,
  projectId,
  entityCopy,
  onIsOpenFilterChange,
}: Props): JSX.Element => {
  const columns: V1ProjectColumn[] = Loadable.getOrElse([], loadableColumns).filter(
    (column) => !bannedFilterColumns?.has(column.column),
  );
  const fieldCount = useObservable(formStore.fieldCount);
  const formset = useObservable(formStore.formset);

  const handleIsOpenFilterChange = useCallback(
    (newOpen: boolean) => {
      if (newOpen) {
        Loadable.match(formset, {
          _: () => {
            return;
          },
          Loaded: (data) => {
            // if there's no conditions, add default condition
            if (data.filterGroup.children.length === 0) {
              formStore.addChild(data.filterGroup.id, FormKind.Field);
            }
          },
        });
      }
      onIsOpenFilterChange?.(newOpen);
    },
    [formStore, formset, onIsOpenFilterChange],
  );

  const onHidePopOver = () => onIsOpenFilterChange?.(false);

  return (
    <div>
      <Dropdown
        content={
          <div
            onKeyDown={(e) => {
              if (e.key === 'Escape') {
                onHidePopOver();
              }
            }}
            onMouseMove={(e) => e.stopPropagation()}>
            <FilterForm
              columns={columns}
              entityCopy={entityCopy}
              formStore={formStore}
              projectId={projectId}
              onHidePopOver={onHidePopOver}
            />
          </div>
        }
        open={isOpenFilter}
        onOpenChange={handleIsOpenFilterChange}>
        <Button
          data-test-component="tableFilter"
          hideChildren={isMobile}
          icon={<Icon decorative name="filter" />}>
          Filter {fieldCount > 0 && `(${fieldCount})`}
        </Button>
      </Dropdown>
    </div>
  );
};

export default TableFilter;
