import { FilterOutlined } from '@ant-design/icons';
import { Button, Popover } from 'antd';
import { useState } from 'react';

import FilterForm from './components/FilterForm';
import { FilterFormStore } from './components/FilterFormStore';

const TableFilter = (): JSX.Element => {
  const [formStore] = useState<FilterFormStore>(() => new FilterFormStore());
  const [open, setOpen] = useState(false);

  const handleOpenChange = (newOpen: boolean) => {
    setOpen(newOpen);
  };

  return (
    <div>
      <Popover
        content={<FilterForm formStore={formStore} />}
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
