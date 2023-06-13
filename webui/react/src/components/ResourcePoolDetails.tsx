import { Divider, Modal } from 'antd';
import React, { Fragment } from 'react';

import Json from 'components/Json';
import { ResourcePool } from 'types';
import { clone } from 'utils/data';
import { camelCaseToSentence } from 'utils/string';

import { PoolLogo } from './ResourcePoolCard';
import css from './ResourcePoolDetails.module.scss';

interface Props {
  finally?: () => void;
  resourcePool: ResourcePool;
  visible: boolean;
}

const ResourcePoolDetails: React.FC<Props> = ({ resourcePool: pool, ...props }: Props) => {
  const details = clone(pool.details);
  for (const key in details) {
    if (details[key] === null) {
      delete details[key];
    }
  }

  const mainSection = clone(pool);
  delete mainSection.details;

  const title = (
    <div>
      <PoolLogo type={pool.type} />
      {' ' + pool.name}
    </div>
  );

  return (
    <Modal
      cancelButtonProps={{ style: { display: 'none' } }}
      cancelText=""
      mask
      open={props.visible}
      style={{ minWidth: '600px' }}
      title={title}
      onCancel={props.finally}
      onOk={props.finally}>
      <Json json={mainSection} translateLabel={camelCaseToSentence} />
      {Object.keys(details).map((key) => (
        <Fragment key={key}>
          <Divider />
          <div className={css.subTitle}>{camelCaseToSentence(key)}</div>
          <Json json={details[key]} translateLabel={camelCaseToSentence} />
        </Fragment>
      ))}
    </Modal>
  );
};

export default ResourcePoolDetails;
