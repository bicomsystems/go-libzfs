/* C wrappers around some zfs calls and C in general that should simplify
 * using libzfs from go language, make go code shorter and more readable.
 */

#ifndef SERVERWARE_ZFS_H
#define SERVERWARE_ZFS_H

struct dataset_list {
	zfs_handle_t *zh;
	void *pnext;
};

typedef struct dataset_list dataset_list_t;
typedef struct dataset_list* dataset_list_ptr;


dataset_list_t *create_dataset_list_item();
void dataset_list_close(dataset_list_t *list);
void dataset_list_free(dataset_list_t *list);

dataset_list_t* dataset_list_root();
dataset_list_t* dataset_list_children(dataset_list_t *dataset);
dataset_list_t *dataset_next(dataset_list_t *dataset);
int dataset_type(dataset_list_ptr dataset);

dataset_list_ptr dataset_open(const char *path);
int dataset_create(const char *path, zfs_type_t type, nvlist_ptr props);
int dataset_destroy(dataset_list_ptr dataset, boolean_t defer);
zpool_list_ptr dataset_get_pool(dataset_list_ptr dataset);
int dataset_prop_set(dataset_list_ptr dataset, zfs_prop_t prop, const char *value);
int dataset_user_prop_set(dataset_list_ptr dataset, const char *prop, const char *value);
int dataset_clone(dataset_list_ptr dataset, const char *target, nvlist_ptr props);
int dataset_snapshot(const char *path, boolean_t recur, nvlist_ptr props);
int dataset_rollback(dataset_list_ptr dataset, dataset_list_ptr snapshot, boolean_t force);
int dataset_promote(dataset_list_ptr dataset);
int dataset_rename(dataset_list_ptr dataset, const char* new_name, boolean_t recur, boolean_t force_unm);
const char* dataset_is_mounted(dataset_list_ptr dataset);
int dataset_mount(dataset_list_ptr dataset, const char *options, int flags);
int dataset_unmount(dataset_list_ptr dataset, int flags);
int dataset_unmountall(dataset_list_ptr dataset, int flags);
const char *dataset_get_name(dataset_list_ptr ds);

property_list_t *read_dataset_property(dataset_list_t *dataset, int prop);
property_list_t *read_user_property(dataset_list_t *dataset, const char* prop);

char** alloc_cstrings(int size);
void strings_setat(char **a, int at, char *v);

#endif
/* SERVERWARE_ZFS_H */
