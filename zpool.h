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
nvlist_t *nvlist_array_at(nvlist_t **a, uint_t i);

int nvlist_lookup_uint64_array_vds(nvlist_t *nv, const char *p,
	vdev_stat_t **vds, uint_t *c);

int nvlist_lookup_uint64_array_ps(nvlist_t *nv, const char *p,
	pool_scan_stat_t **vds, uint_t *c);

int refresh_stats(zpool_list_t *pool);

char *sZPOOL_CONFIG_VERSION;
char *sZPOOL_CONFIG_POOL_NAME;
char *sZPOOL_CONFIG_POOL_STATE;
char *sZPOOL_CONFIG_POOL_TXG;
char *sZPOOL_CONFIG_POOL_GUID;
char *sZPOOL_CONFIG_CREATE_TXG;
char *sZPOOL_CONFIG_TOP_GUID;
char *sZPOOL_CONFIG_VDEV_TREE;
char *sZPOOL_CONFIG_TYPE;
char *sZPOOL_CONFIG_CHILDREN;
char *sZPOOL_CONFIG_ID;
char *sZPOOL_CONFIG_GUID;
char *sZPOOL_CONFIG_PATH;
char *sZPOOL_CONFIG_DEVID;
char *sZPOOL_CONFIG_METASLAB_ARRAY;
char *sZPOOL_CONFIG_METASLAB_SHIFT;
char *sZPOOL_CONFIG_ASHIFT;
char *sZPOOL_CONFIG_ASIZE;
char *sZPOOL_CONFIG_DTL;
char *sZPOOL_CONFIG_SCAN_STATS;
char *sZPOOL_CONFIG_VDEV_STATS;
char *sZPOOL_CONFIG_WHOLE_DISK;
char *sZPOOL_CONFIG_ERRCOUNT;
char *sZPOOL_CONFIG_NOT_PRESENT;
char *sZPOOL_CONFIG_SPARES;
char *sZPOOL_CONFIG_IS_SPARE;
char *sZPOOL_CONFIG_NPARITY;
char *sZPOOL_CONFIG_HOSTID;
char *sZPOOL_CONFIG_HOSTNAME;
char *sZPOOL_CONFIG_LOADED_TIME;
char *sZPOOL_CONFIG_UNSPARE;
char *sZPOOL_CONFIG_PHYS_PATH;
char *sZPOOL_CONFIG_IS_LOG;
char *sZPOOL_CONFIG_L2CACHE;
char *sZPOOL_CONFIG_HOLE_ARRAY;
char *sZPOOL_CONFIG_VDEV_CHILDREN;
char *sZPOOL_CONFIG_IS_HOLE;
char *sZPOOL_CONFIG_DDT_HISTOGRAM;
char *sZPOOL_CONFIG_DDT_OBJ_STATS;
char *sZPOOL_CONFIG_DDT_STATS;
char *sZPOOL_CONFIG_SPLIT;
char *sZPOOL_CONFIG_ORIG_GUID;
char *sZPOOL_CONFIG_SPLIT_GUID;
char *sZPOOL_CONFIG_SPLIT_LIST;
char *sZPOOL_CONFIG_REMOVING;
char *sZPOOL_CONFIG_RESILVER_TXG;
char *sZPOOL_CONFIG_COMMENT;
char *sZPOOL_CONFIG_SUSPENDED;
char *sZPOOL_CONFIG_TIMESTAMP;
char *sZPOOL_CONFIG_BOOTFS;
char *sZPOOL_CONFIG_MISSING_DEVICES;
char *sZPOOL_CONFIG_LOAD_INFO;
char *sZPOOL_CONFIG_REWIND_INFO;
char *sZPOOL_CONFIG_UNSUP_FEAT;
char *sZPOOL_CONFIG_ENABLED_FEAT;
char *sZPOOL_CONFIG_CAN_RDONLY;
char *sZPOOL_CONFIG_FEATURES_FOR_READ;
char *sZPOOL_CONFIG_FEATURE_STATS;
char *sZPOOL_CONFIG_ERRATA;
char *sZPOOL_CONFIG_OFFLINE;
char *sZPOOL_CONFIG_FAULTED;
char *sZPOOL_CONFIG_DEGRADED;
char *sZPOOL_CONFIG_REMOVED;
char *sZPOOL_CONFIG_FRU;
char *sZPOOL_CONFIG_AUX_STATE;
char *sZPOOL_REWIND_POLICY;
char *sZPOOL_REWIND_REQUEST;
char *sZPOOL_REWIND_REQUEST_TXG;
char *sZPOOL_REWIND_META_THRESH;
char *sZPOOL_REWIND_DATA_THRESH;
char *sZPOOL_CONFIG_LOAD_TIME;
char *sZPOOL_CONFIG_LOAD_DATA_ERRORS;
char *sZPOOL_CONFIG_REWIND_TIME;


#endif
/* SERVERWARE_ZPOOL_H */
