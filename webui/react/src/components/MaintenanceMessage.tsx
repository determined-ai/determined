import React, { useEffect } from 'react';

import { MaintenanceMessage } from 'stores/determinedInfo';

import css from './MaintenanceMessage.module.scss';

interface Props {
	message?: MaintenanceMessage;
}

const MaintenanceMessage: React.FC<Props> = ({ message }) => {
	useEffect(() => {
		console.log('message component', message);
	}, [message]);

	const msg = message ? message.message + "adding a bunch of text to see what it will be like with a longer maintenance message. Adding even more to test extremely long ones. Adding even more to test extremely long ones. Adding even more to test extremely long ones. Adding even more to test extremely long ones. Adding even more to test extremely long ones. Adding even more to test extremely long ones. Adding even more to test extremely long ones. Adding even more to test extremely long ones." : "";
	const trimmedMsg = msg.substring(0, 250);

	return message ? (
		<div className={css.base}>
			<span className={css.maintenanceMessageLabel}>Maintenance Message</span>: { trimmedMsg }
		</div>
	) : null;
};

export default MaintenanceMessage;
