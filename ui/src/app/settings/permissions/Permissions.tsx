import { PlusIcon } from '@heroicons/react/24/outline';
import { useRef, useState } from 'react';
import { useSelector } from 'react-redux';
import { selectReadonly } from '~/app/meta/metaSlice';
import Slideover from '~/components/Slideover';
import Button from '~/components/forms/buttons/Button';
import SetupPermissionsForm from '~/components/permissions/SetupPermissionsForm';

export default function Permissions() {
  const [showPermissionsForm, setShowPermissionsForm] =
    useState<boolean>(false);

  const readOnly = useSelector(selectReadonly);
  const permissionsFormRef = useRef(null);
  return (
    <>
      {/* permisisons edit form */}
      <Slideover
        open={showPermissionsForm}
        setOpen={setShowPermissionsForm}
        ref={permissionsFormRef}
      >
        <SetupPermissionsForm
          ref={permissionsFormRef}
          setOpen={setShowPermissionsForm}
          onSuccess={() => {
            setShowPermissionsForm(false);
          }}
        />
      </Slideover>

      <div className="my-10">
        <div className="sm:flex sm:items-center">
          <div className="sm:flex-auto">
            <h3 className="text-gray-700 text-xl font-semibold">Permissions</h3>
            <p className="text-gray-500 mt-2 text-sm">
              Permissions are managed at the namespace level.
            </p>
            <p className="text-gray-500 text-sm">
              You can assign roles and actions to a namespace to control what
              users can do within that namespace.
            </p>
          </div>
          <div className="mt-4">
            <Button
              variant="primary"
              disabled={readOnly}
              title={readOnly ? 'Not allowed in Read-Only mode' : undefined}
              onClick={() => {
                setShowPermissionsForm(true);
              }}
            >
              <PlusIcon
                className="text-white -ml-1.5 mr-1 h-5 w-5"
                aria-hidden="true"
              />
              <span>Setup Permissions</span>
            </Button>
          </div>
        </div>
      </div>
    </>
  );
}
