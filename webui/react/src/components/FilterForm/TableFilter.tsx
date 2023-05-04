import { FilterOutlined } from '@ant-design/icons';
import { Button, Popover } from 'antd';
import { useMemo, useState } from 'react';

import { V1ProjectColumn } from 'services/api-ts-sdk';
import { Loadable } from 'utils/loadable';

import FilterForm from './components/FilterForm';
import { FilterFormStore } from './components/FilterFormStore';

interface Props {
  loadableColumns: Loadable<V1ProjectColumn[]>;
  formStore: FilterFormStore;
}

const TableFilter = ({ loadableColumns, formStore }: Props): JSX.Element => {
  const [open, setOpen] = useState(false);

  const columns: V1ProjectColumn[] = useMemo(() => {
    return Loadable.getOrElse([], loadableColumns);
  }, [loadableColumns]);

  const handleOpenChange = (newOpen: boolean) => {
    setOpen(newOpen);
  };

  const onHidePopOver = () => {
    setOpen(false);
  };

  return (
    <div>
      <Popover
        content={
          <FilterForm columns={columns} formStore={formStore} onHidePopOver={onHidePopOver} />
        }
        open={open}
        placement="bottomLeft"
        trigger="click"
        onOpenChange={handleOpenChange}>
        <Button icon={<FilterOutlined />}>Filter</Button>
      </Popover>
    </div>
  );
};

export default TableFilter;
