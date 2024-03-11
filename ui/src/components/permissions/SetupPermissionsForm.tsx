import { Dialog } from '@headlessui/react';
import { XMarkIcon } from '@heroicons/react/24/outline';
import { Form, Formik } from 'formik';
import Loading from '~/components/Loading';
import MoreInfo from '~/components/MoreInfo';
import Button from '~/components/forms/buttons/Button';
import { forwardRef, useState } from 'react';
import * as Yup from 'yup';
import { useSelector } from 'react-redux';
import { selectNamespaces } from '~/app/namespacesSlice';
import { SelectableNamespace } from '~/components/namespaces/NamespaceListbox';
import Listbox from '~/components/forms/Listbox';

const permissionsValidationSchema = Yup.object({});

type SetupPermissionsFormProps = {
  setOpen: (open: boolean) => void;
  onSuccess: () => void;
};

const SetupPermissionsForm = forwardRef(
  (props: SetupPermissionsFormProps, ref: any) => {
    const { setOpen, onSuccess } = props;

    const namespaces = useSelector(selectNamespaces);

    const [selectedNamespace, setSelectedNamespace] =
      useState<SelectableNamespace>({
        ...namespaces[0],
        displayValue: namespaces[0].name
      });

    return (
      <Formik
        initialValues={{}}
        onSubmit={(values, { setSubmitting }) => {
          //handleSubmit(values);
          //   .then(() => {
          //     clearError();
          //     setSuccess(
          //       `Successfully ${submitPhrase.toLocaleLowerCase()}d namespace.`
          //     );
          //     onSuccess();
          //   })
          //   .catch((err) => {
          //     setError(err);
          //   })
          //   .finally(() => {
          //     setSubmitting(false);
          //});
        }}
        validationSchema={permissionsValidationSchema}
      >
        {(formik) => (
          <Form className="bg-white flex h-full flex-col overflow-y-scroll shadow-xl">
            <div className="flex-1">
              <div className="bg-gray-50 px-4 py-6 sm:px-6">
                <div className="flex items-start justify-between space-x-3">
                  <div className="space-y-1">
                    <Dialog.Title className="text-gray-900 text-lg font-medium">
                      Set Permissions
                    </Dialog.Title>
                    <MoreInfo href="https://www.flipt.io/docs/concepts#permissions">
                      Learn more about permissions
                    </MoreInfo>
                  </div>
                  <div className="flex h-7 items-center">
                    <button
                      type="button"
                      className="text-gray-400 hover:text-gray-500"
                      onClick={() => setOpen(false)}
                    >
                      <span className="sr-only">Close panel</span>
                      <XMarkIcon className="h-6 w-6" aria-hidden="true" />
                    </button>
                  </div>
                </div>
              </div>
              <div className="space-y-6 py-6 sm:space-y-0 sm:divide-y sm:divide-gray-200 sm:py-0">
                <div className="space-y-1 px-4 sm:grid sm:grid-cols-3 sm:gap-4 sm:space-y-0 sm:px-6 sm:py-5">
                  <div>
                    <label
                      htmlFor="namespaceKey"
                      className="text-gray-900 block text-sm font-medium sm:mt-px sm:pt-2"
                    >
                      Namespace
                    </label>
                  </div>
                  <div className="sm:col-span-2">
                    <Listbox<SelectableNamespace>
                      id="namespaceKey"
                      name="namespaceKey"
                      values={namespaces.map((n) => ({
                        ...n,
                        displayValue: n.name
                      }))}
                      selected={{
                        ...selectedNamespace,
                        displayValue: selectedNamespace?.name || ''
                      }}
                      setSelected={setSelectedNamespace}
                    />
                  </div>
                </div>
              </div>
              <div className="border-gray-200 flex-shrink-0 border-t px-4 py-5 sm:px-6">
                <div className="flex justify-end space-x-3">
                  <Button onClick={() => setOpen(false)}>Cancel</Button>
                  <Button
                    variant="primary"
                    className="min-w-[80px]"
                    type="submit"
                    disabled={
                      !(formik.dirty && formik.isValid && !formik.isSubmitting)
                    }
                  >
                    {formik.isSubmitting ? <Loading isPrimary /> : 'Create'}
                  </Button>
                </div>
              </div>
            </div>
          </Form>
        )}
      </Formik>
    );
  }
);

SetupPermissionsForm.displayName = 'PermissionsForm';
export default SetupPermissionsForm;
