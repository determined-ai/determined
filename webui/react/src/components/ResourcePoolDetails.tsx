import { Divider, Modal } from 'antd';
import React from 'react';

import Json from 'components/Json';
import { ResourcePool } from 'types';
import { clone } from 'utils/data';
import { camelCaseToSentence } from 'utils/string';

import { rpLogo } from './ResourcePoolCard';

interface Props {
  finally: () => void;
  resourcePool: ResourcePool;
  visible: boolean;
}

const ResourcePoolDetails: React.FC<Props> = ({ resourcePool: rp, ...props }: Props) => {
  const details = clone(rp.details);
  const providerSpecific = details[rp.type];
  delete details[rp.type];

  const title = (
    <div>
      {rpLogo(rp.type)}
      {' ' + rp.name}
    </div>
  );

  return (
    <Modal
      cancelButtonProps={{ style: { display:'none' } }}
      cancelText=""
      mask
      style={{ minWidth: '60rem' }}
      title={title}
      visible={props.visible}
      onCancel={props.finally}
      onOk={props.finally}
    >
      <Json json={details} transformLabel={camelCaseToSentence} />
      {providerSpecific &&
      <>
        <Divider />
        <Json json={providerSpecific} transformLabel={camelCaseToSentence} />
      </>
      }
    </Modal>
  );

};

export default ResourcePoolDetails;
