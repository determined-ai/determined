import { DownOutlined, LeftOutlined } from '@ant-design/icons';
import { Breadcrumb, Button, Dropdown, Menu } from 'antd';
import React from 'react';

import Icon from 'components/Icon';
import { ModelItem } from 'types';

interface Props {
  model: ModelItem;
}

const ModelHeader: React.FC<Props> = ({ model }: Props) => {
  return (
    <header style={{
      backgroundColor: 'var(--theme-colors-monochrome-17)',
      borderBottom: '1px solid var(--theme-colors-monochrome-12)',
      margin: 0,
      padding: 12,
    }}>
      <div style={{
        borderBottom: '1px solid var(--theme-colors-monochrome-12)',
        paddingBottom: 8,
      }}>
        <Breadcrumb separator="">
          <Breadcrumb.Item href="det/models">
            <LeftOutlined style={{ marginRight: 10 }} />
          </Breadcrumb.Item>
          <Breadcrumb.Item href="det/models">Model Registry</Breadcrumb.Item>
          <Breadcrumb.Separator />
          <Breadcrumb.Item>Model 12</Breadcrumb.Item>
        </Breadcrumb>
      </div>
      <div style={{
        alignItems: 'center',
        display: 'flex',
        justifyContent: 'space-between',
        marginLeft: 24,
        marginRight: 24,
      }}>
        <div>
          <img />
          <h1>{model.name}</h1>
        </div>
        <div style={{ display: 'flex', gap: 4 }}>
          <Dropdown overlay={<Menu />}>
            <Button>
              Open in <DownOutlined />
            </Button>
          </Dropdown>
          <Button>Download Model</Button>
          <Button><Icon name="overflow-horizontal" size="tiny" /></Button>
        </div>
      </div>
    </header>
  );
};

export default ModelHeader;
