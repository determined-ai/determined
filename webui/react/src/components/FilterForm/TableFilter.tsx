import { Popover } from 'antd';
import { useObservable } from 'micro-observables';

import FilterForm from 'components/FilterForm/components/FilterForm';
import { FilterFormStore } from 'components/FilterForm/components/FilterFormStore';
import Button from 'components/kit/Button';
import Icon from 'components/kit/Icon';
import { V1ProjectColumn } from 'services/api-ts-sdk';
import { Loadable } from 'utils/loadable';

interface Props {
  loadableColumns: Loadable<V1ProjectColumn[]>;
  formStore: FilterFormStore;
  setIsOpenFilter: (value: boolean) => void;
  isOpenFilter: boolean;
}

const TableFilter = ({
  loadableColumns,
  formStore,
  isOpenFilter,
  setIsOpenFilter,
}: Props): JSX.Element => {
  const columns: V1ProjectColumn[] = Loadable.getOrElse([], loadableColumns);
  const fieldCount = useObservable(formStore.fieldCount);

  const onIsOpenFilterChange = (newOpen: boolean) => {
    setIsOpenFilter(newOpen);
  };

  const onHidePopOver = () => {
    setIsOpenFilter(false);
  };

  return (
    <div>
      <Popover
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
        destroyTooltipOnHide
        open={isOpenFilter}
        placement="bottomLeft"
        trigger="click"
        onOpenChange={onIsOpenFilterChange}>
        <Button icon={<Icon name="filter" title="filter" />}>
          Filter{fieldCount > 0 && <span>({fieldCount})</span>}
        </Button>
      </Popover>
    </div>
  );
};

export default TableFilter;
