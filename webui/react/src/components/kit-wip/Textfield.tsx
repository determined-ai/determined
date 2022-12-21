import { Input } from 'antd';
import React, { CSSProperties, ReactNode } from 'react';

interface TextfieldProps {
}

const Textfield: React.FC<TextfieldProps> = (props: TextfieldProps) => {
  return (
    <Input {...props} />
  );
};

//Input.TextArea
//Input.Group
//Input.Password

export default Textfield;
