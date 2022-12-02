import { ComponentStory, Meta } from '@storybook/react';
import { Tree } from 'antd';
import type { DataNode, TreeProps } from 'antd/es/tree';
import React, { useState } from 'react';

export default {
  argTypes: {
    type: {
      control: {
        options: ['success', 'info', 'warning', 'error'],
        type: 'inline-radio',
      },
    },
  },
  component: Tree,
  title: 'Ant Design/Tree',
} as Meta<typeof Tree>;

const treeData: DataNode[] = [
  {
    children: [
      {
        children: [
          {
            disableCheckbox: true,
            key: '0-0-0-0',
            title: 'leaf',
          },
          {
            key: '0-0-0-1',
            title: 'leaf',
          },
        ],
        disabled: true,
        key: '0-0-0',
        title: 'parent 1-0',
      },
      {
        children: [{ key: '0-0-1-0', title: <span style={{ color: '#1890ff' }}>sss</span> }],
        key: '0-0-1',
        title: 'parent 1-1',
      },
    ],
    key: '0-0',
    title: 'parent 1',
  },
];

export const Default: ComponentStory<typeof Tree> = (args) => {
  const [gData, setGData] = useState(treeData);

  // This onDrop implementation comes from antd docs, would require refactoring before using in production code.
  /* eslint-disable */
  const onDrop: TreeProps['onDrop'] = (info: any) => {
    const dropKey = info.node.key;
    const dragKey = info.dragNode.key;
    const dropPos = info.node.pos.split('-');
    const dropPosition = info.dropPosition - Number(dropPos[dropPos.length - 1]);

    const loop = (
      data: DataNode[],
      key: React.Key,
      callback: (node: DataNode, i: number, data: DataNode[]) => void,
    ) => {
      for (let i = 0; i < data.length; i++) {
        if (data[i].key === key) {
          return callback(data[i], i, data);
        }
        if (data[i].children) {
          loop(data[i].children!, key, callback);
        }
      }
    };
    const data = [...gData];

    // Find dragObject
    let dragObj: DataNode;
    loop(data, dragKey, (item, index, arr) => {
      arr.splice(index, 1);
      dragObj = item;
    });

    if (!info.dropToGap) {
      // Drop on the content
      loop(data, dropKey, (item) => {
        item.children = item.children || [];
        item.children.unshift(dragObj);
      });
    } else if (
      (info.node.children || []).length > 0 && // Has children
      info.node.expanded && // Is expanded
      dropPosition === 1 // On the bottom gap
    ) {
      loop(data, dropKey, (item) => {
        item.children = item.children || [];
        item.children.unshift(dragObj);
      });
    } else {
      let ar: DataNode[] = [];
      let i: number;
      loop(data, dropKey, (_item, index, arr) => {
        ar = arr;
        i = index;
      });
      if (dropPosition === -1) {
        ar.splice(i!, 0, dragObj!);
      } else {
        ar.splice(i! + 1, 0, dragObj!);
      }
    }
    setGData(data);
  };
  /* eslint-enable */

  return (
    <Tree
      treeData={gData}
      {...args}
      defaultCheckedKeys={['0-0-0', '0-0-1']}
      defaultExpandedKeys={['0-0-0', '0-0-1']}
      defaultSelectedKeys={['0-0-0', '0-0-1']}
      onDrop={onDrop}
    />
  );
};

Default.args = {
  blockNode: false,
  checkable: true,
  draggable: false,
};
