import React from 'react';

import Button from 'components/kit/Button';
import Tooltip from 'components/kit/Tooltip';
import useModalJupyterLab from 'hooks/useModal/JupyterLab/useModalJupyterLab';

interface Props {
  enabled?: boolean;
}

const JupyterLabButton: React.FC<Props> = ({ enabled }: Props) => {
  const { contextHolder: modalJupyterLabContextHolder, modalOpen: openJupyterLabModal } =
    useModalJupyterLab({});

  return (
    <>
      {enabled ? (
        <Button onClick={() => openJupyterLabModal()}>Launch JupyterLab</Button>
      ) : (
        <Tooltip placement="leftBottom" title="You do not have permission to launch JupyterLab">
          <div>
            <Button disabled>Launch JupyterLab</Button>
          </div>
        </Tooltip>
      )}
      {modalJupyterLabContextHolder}
    </>
  );
};

export default JupyterLabButton;
