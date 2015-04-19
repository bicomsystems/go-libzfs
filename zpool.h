/* C wrappers around some zfs calls and C in general that should simplify
 * using libzfs from go language, make go code shorter and more readable.
 */

#ifndef SERVERWARE_ZPOOL_H
#define SERVERWARE_ZPOOL_H

#define INT_MAX_NAME 256
#define INT_MAX_VALUE 1024

struct zpool_list {
	zpool_handle_t *zph;
	void *pnext;
};

typedef struct property_list {
	char value[INT_MAX_VALUE];
	char source[INT_MAX_NAME];
	int property;
	void *pnext;
} property_list_t;

typedef struct zpool_list zpool_list_t;

property_list_t *new_property_list();

zpool_list_t *create_zpool_list_item();
void zprop_source_tostr(char *dst, zprop_source_t source);

zpool_list_t* zpool_list_open(libzfs_handle_t *libzfs, const char *name);
int zpool_list(libzfs_handle_t *libzfs, zpool_list_t **first);
zpool_list_t *zpool_next(zpool_list_t *pool);

void zpool_list_close(zpool_list_t *pool);

int read_zpool_property(zpool_handle_t *zh, property_list_t *list, int prop);
property_list_t *read_zpool_properties(zpool_handle_t *zh);
property_list_t *next_property(property_list_t *list);
void free_properties(property_list_t *root);

pool_state_t zpool_read_state(zpool_handle_t *zh);


const char *lasterr(void);

int
add_prop_list(const char *propname, char *propval, nvlist_t **props,
    boolean_t poolprop);

nvlist_t** nvlist_alloc_array(int count);
void nvlist_array_set(nvlist_t** a, int i, nvlist_t *item);
void nvlist_free_array(nvlist_t **a);


void free_cstring(char *str);

#endif
/* SERVERWARE_ZPOOL_H */
