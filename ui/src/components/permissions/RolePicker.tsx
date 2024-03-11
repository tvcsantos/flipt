import { useRef, useState, useEffect } from 'react';
import { FilterableRole, IRole } from '~/types/Role';
import Combobox from '../forms/Combobox';
import { cls, truncateKey } from '~/utils/helpers';
import { MinusSmallIcon, PlusSmallIcon } from '@heroicons/react/24/outline';

const availableRoles = {
  admin: 'Admin',
  editor: 'Editor',
  viewer: 'Viewer'
};

type RolePickerProps = {
  readonly?: boolean;
  roles: IRole[];
  selectedRoles: FilterableRole[];
  roleAdd: (role: FilterableRole) => void;
  roleReplace: (index: number, role: FilterableRole) => void;
  roleRemove: (index: number) => void;
};

export default function RolePicker({
  readonly = false,
  roles,
  selectedRoles: parentRoles,
  roleAdd,
  roleReplace,
  roleRemove
}: RolePickerProps) {
  const rolesSet = useRef<Set<string>>(
    new Set<string>(parentRoles.map((s) => s.id))
  );

  const [editing, setEditing] = useState<boolean>(true);

  useEffect(() => {
    setEditing(true);
  }, [parentRoles]);

  const handleRoleRemove = (index: number) => {
    const filterableRole = parentRoles[index];

    // Remove references to the role that is being deleted.
    rolesSet.current!.delete(filterableRole.id);
    roleRemove(index);

    if (editing && parentRoles.length == 1) {
      setEditing(true);
    }
  };

  const handleRoleSelected = (index: number, role: FilterableRole | null) => {
    if (!role) {
      return;
    }

    const selectedRoleList = [...parentRoles];
    const roleSetCurrent = rolesSet.current!;

    if (index <= parentRoles.length - 1) {
      const previousRole = selectedRoleList[index];
      if (roleSetCurrent.has(previousRole.id)) {
        roleSetCurrent.delete(previousRole.id);
      }

      roleSetCurrent.add(role.id);
      roleReplace(index, role);
    } else {
      roleSetCurrent.add(role.id);
      roleAdd(role);
    }

    setEditing(false);
  };
  return (
    <div className="space-y-2">
      {parentRoles.map((selectedRole, index) => (
        <div className="flex w-full space-x-1" key={index}>
          <div className="w-5/6">
            <Combobox<FilterableRole>
              id={`roleKey-${index}`}
              name={`roleKey-${index}`}
              placeholder="Select or search for a role"
              values={roles
                .filter((s) => !rolesSet.current!.has(s.key))
                .map((s) => ({
                  ...s,
                  filterValue: truncateKey(s.key),
                  displayValue: s.name
                }))}
              disabled={readonly}
              selected={selectedRole}
              setSelected={(filterableRole) => {
                handleRoleSelected(index, filterableRole);
              }}
              inputClassName={
                readonly
                  ? 'cursor-not-allowed bg-gray-100 text-gray-500'
                  : undefined
              }
            />
          </div>
          {editing && parentRoles.length - 1 === index ? (
            <div>
              <button
                type="button"
                className={cls('text-gray-400 mt-2 hover:text-gray-500', {
                  'hover:text-gray-400': readonly
                })}
                onClick={() => setEditing(false)}
                title={readonly ? 'Not allowed in Read-Only mode' : undefined}
                disabled={readonly}
              >
                <PlusSmallIcon className="h-6 w-6" aria-hidden="true" />
              </button>
            </div>
          ) : (
            <div>
              <button
                type="button"
                className="text-gray-400 mt-2 hover:text-gray-500"
                onClick={() => handleRoleRemove(index)}
                title={readonly ? 'Not allowed in Read-Only mode' : undefined}
                disabled={readonly}
              >
                <MinusSmallIcon className="h-6 w-6" aria-hidden="true" />
              </button>
            </div>
          )}
        </div>
      ))}
      {(!editing || parentRoles.length === 0) && (
        <div className="w-full">
          <div className="w-5/6">
            <Combobox<FilterableRole>
              id={`roleKey-${parentRoles.length}`}
              name={`roleKey-${parentRoles.length}`}
              placeholder="Select or search for a role"
              values={roles
                .filter((s) => !rolesSet.current!.has(s.key))
                .map((s) => ({
                  ...s,
                  filterValue: truncateKey(s.key),
                  displayValue: s.name
                }))}
              selected={null}
              setSelected={(filterableRole) => {
                handleRoleSelected(parentRoles.length, filterableRole);
              }}
            />
          </div>
        </div>
      )}
    </div>
  );
}
