import { CopyOutlined } from '@ant-design/icons';
import { Tooltip } from 'antd';
import React, { useCallback, useState } from 'react';

type TextOptions = 'Copy to Clipboard' | 'Copied!'

interface Props {
  handleCopy: () => Promise<void>;
}

const CopyButton: React.FC<Props> = ({ handleCopy }: Props) => {
  const [ text, setText ] = useState<TextOptions>('Copy to Clipboard');

  const handleClick = useCallback(async () => {
    await handleCopy();
    setText('Copied!');
    setTimeout(() => {
      setText('Copy to Clipboard');
    }, 2000);
  }, [ handleCopy ]);

  return (
    <Tooltip title={text}>
      <CopyOutlined onClick={handleClick} />
    </Tooltip>
  );
};

export default CopyButton;
