import { Button, Popover } from 'antd';
import { useState } from 'react';

import FilterForm from 'components/FilterForm/FilterForm';
import { FilterFormStore } from 'components/FilterForm/FilterFormStore';

const formStore = new FilterFormStore();

const TEST = (): JSX.Element => {
  const [open, setOpen] = useState(false);

  const handleOpenChange = (newOpen: boolean) => {
    setOpen(newOpen);
  };

  return (
    <>
      <FilterForm formStore={formStore} />
      <div style={{ display: 'flex', justifyContent: 'center', marginTop: '40px' }}>
        <Popover
          content={<FilterForm formStore={formStore} />}
          open={open}
          placement="bottom"
          trigger="click"
          onOpenChange={handleOpenChange}>
          <Button>Click ME</Button>
        </Popover>
      </div>
    </>
  );
};

export default TEST;
