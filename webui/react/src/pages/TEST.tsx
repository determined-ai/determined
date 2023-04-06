import { FilterOutlined } from '@ant-design/icons';
import { Button, Popconfirm } from 'antd';

import FilterForm from 'components/FilterForm/FilterForm';
import { FormClassStore, formSets } from 'components/FilterForm/FilterFormStore';

const formClassStore = new FormClassStore(formSets);

const TEST = (): JSX.Element => {
  return (
    <>
      <FilterForm formClassStore={formClassStore} />
      <div style={{ display: 'flex', justifyContent: 'center', marginTop: '40px' }}>
        <Popconfirm
          description={<FilterForm formClassStore={formClassStore} />}
          icon={<FilterOutlined />}
          title={'Table Filter'}>
          <Button>Click ME</Button>
        </Popconfirm>
      </div>
    </>
  );
};

export default TEST;
