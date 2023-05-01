import { FilterOutlined } from '@ant-design/icons';
import { Button, Popover } from 'antd';
import { useMemo, useState } from 'react';

import { V1ProjectColumn } from 'services/api-ts-sdk';
import { Loadable } from 'utils/loadable';

import FilterForm from './components/FilterForm';
import { FilterFormStore } from './components/FilterFormStore';

interface Props {
  loadableColumns: Loadable<V1ProjectColumn[]>;
}

const TableFilter = ({ loadableColumns }: Props): JSX.Element => {
  const [formStore] = useState<FilterFormStore>(() => new FilterFormStore());
  const [open, setOpen] = useState(false);

  const columns: V1ProjectColumn[] = useMemo(() => {
    return Loadable.getOrElse([], loadableColumns);
  }, [loadableColumns]);

  const handleOpenChange = (newOpen: boolean) => {
    setOpen(newOpen);
  };

  return (
    <div>
      <Popover
        content={<FilterForm columns={columns} formStore={formStore} />}
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
