import { useObservable } from 'micro-observables';
import { useCallback } from 'react';

import FilterForm from 'components/FilterForm/components/FilterForm';
import { FilterFormStore } from 'components/FilterForm/components/FilterFormStore';
import { FormKind } from 'components/FilterForm/components/type';
import Button from 'components/kit/Button';
import Dropdown from 'components/kit/Dropdown';
import Icon from 'components/kit/Icon';
import { V1ProjectColumn } from 'services/api-ts-sdk';
import { Loadable } from 'utils/loadable';

interface Props {
  loadableColumns: Loadable<V1ProjectColumn[]>;
  formStore: FilterFormStore;
  setIsOpenFilter: (value: boolean) => void;
  isMobile?: boolean;
  isOpenFilter: boolean;
}

const TableFilter = ({
  loadableColumns,
  formStore,
  isMobile = false,
  isOpenFilter,
  setIsOpenFilter,
}: Props): JSX.Element => {
  const columns: V1ProjectColumn[] = Loadable.getOrElse([], loadableColumns);
  const fieldCount = useObservable(formStore.fieldCount);
  const formset = useObservable(formStore.formset);

  const onIsOpenFilterChange = useCallback(
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
      setIsOpenFilter(newOpen);
    },
    [formStore, formset, setIsOpenFilter],
  );

  const onHidePopOver = () => {
    setIsOpenFilter(false);
  };

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
        onOpenChange={onIsOpenFilterChange}>
        <Button hideChildren={isMobile} icon={<Icon decorative name="filter" />}>
          Filter {fieldCount > 0 && `(${fieldCount})`}
        </Button>
      </Dropdown>
    </div>
  );
};

export default TableFilter;
