import { useObservable } from 'micro-observables';
import { useCallback } from 'react';

import FilterForm from 'components/FilterForm/components/FilterForm';
import { FilterFormStore } from 'components/FilterForm/components/FilterFormStore';
import { FormKind } from 'components/FilterForm/components/type';
import Button from 'components/kit/Button';
import Dropdown from 'components/kit/Dropdown';
import Icon from 'components/kit/Icon';
import { Loadable } from 'components/kit/utils/loadable';
import { V1ProjectColumn } from 'services/api-ts-sdk';

interface Props {
  loadableColumns: Loadable<V1ProjectColumn[]>;
  formStore: FilterFormStore;
  isMobile?: boolean;
  isOpenFilter: boolean;
  onIsOpenFilterChange?: (value: boolean) => void;
}

const TableFilter = ({
  loadableColumns,
  formStore,
  isMobile = false,
  isOpenFilter,
  onIsOpenFilterChange,
}: Props): JSX.Element => {
  const columns: V1ProjectColumn[] = Loadable.getOrElse([], loadableColumns);
  const fieldCount = useObservable(formStore.fieldCount);
  const formset = useObservable(formStore.formset);

  const handleIsOpenFilterChange = useCallback(
    (newOpen: boolean) => {
      if (newOpen) {
        Loadable.match(formset, {
          Loaded: (data) => {
            // if there's no conditions, add default condition
            if (data.filterGroup.children.length === 0) {
              formStore.addChild(data.filterGroup.id, FormKind.Field);
            }
          },
          NotLoaded: () => {
            return;
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
            }}>
            <FilterForm columns={columns} formStore={formStore} onHidePopOver={onHidePopOver} />
          </div>
        }
        open={isOpenFilter}
        onOpenChange={handleIsOpenFilterChange}>
        <Button hideChildren={isMobile} icon={<Icon decorative name="filter" />}>
          Filter {fieldCount > 0 && `(${fieldCount})`}
        </Button>
      </Dropdown>
    </div>
  );
};

export default TableFilter;
