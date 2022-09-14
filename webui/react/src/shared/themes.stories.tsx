import { Button } from 'antd';
import React from 'react';

export default { title: 'Shared/Theme' };

export const Default = (): React.ReactNode => <Button>Hello World</Button>;

export const ButtonDisabled = (): React.ReactNode => <Button disabled>Hello World</Button>;
