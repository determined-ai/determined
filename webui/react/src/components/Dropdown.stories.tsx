import { Button, Menu } from 'antd';
import React from 'react';

import Dropdown, { Placement } from './Dropdown';

export default {
  component: Dropdown,
  parameters: { layout: 'centered' },
  title: 'Determined/Dropdowns/Dropdown',
};

const content = (
  <Menu
    items={new Array(7).fill(null).map((_, index) => ({ key: index, label: `Menu Item ${index}` }))}
  />
);

export const Default = (): React.ReactNode => (
  <Dropdown content={content}>
    <Button>Default Dropdown</Button>
  </Dropdown>
);
export const Placements = (): React.ReactNode => {
  return (
    <table>
      <tbody>
        <tr>
          <td />
          <td>
            <Dropdown content={content} placement={Placement.BottomLeft}>
              <Button>Top Left</Button>
            </Dropdown>
          </td>
          <td align="center">
            <Dropdown content={content} placement={Placement.Top}>
              <Button>Top</Button>
            </Dropdown>
          </td>
          <td align="right">
            <Dropdown content={content} placement={Placement.TopRight}>
              <Button>Top Right</Button>
            </Dropdown>
          </td>
          <td />
        </tr>
        <tr>
          <td>
            <Dropdown content={content} placement={Placement.LeftTop}>
              <Button>Left Top</Button>
            </Dropdown>
          </td>
          <td colSpan={3} />
          <td align="right">
            <Dropdown content={content} placement={Placement.RightTop}>
              <Button>Right Top</Button>
            </Dropdown>
          </td>
        </tr>
        <tr>
          <td>
            <Dropdown content={content} placement={Placement.Left}>
              <Button>Left</Button>
            </Dropdown>
          </td>
          <td colSpan={3} />
          <td align="right">
            <Dropdown content={content} placement={Placement.Right}>
              <Button>Right</Button>
            </Dropdown>
          </td>
        </tr>
        <tr>
          <td>
            <Dropdown content={content} placement={Placement.LeftBottom}>
              <Button>Left Bottom</Button>
            </Dropdown>
          </td>
          <td colSpan={3} />
          <td align="right">
            <Dropdown content={content} placement={Placement.RightBottom}>
              <Button>Right Bottom</Button>
            </Dropdown>
          </td>
        </tr>
        <tr>
          <td />
          <td>
            <Dropdown content={content} placement={Placement.BottomLeft}>
              <Button>Bottom Left</Button>
            </Dropdown>
          </td>
          <td align="center">
            <Dropdown content={content} placement={Placement.Bottom}>
              <Button>Bottom</Button>
            </Dropdown>
          </td>
          <td align="right">
            <Dropdown content={content} placement={Placement.BottomRight}>
              <Button>Bottom Right</Button>
            </Dropdown>
          </td>
          <td />
        </tr>
      </tbody>
    </table>
  );
};
