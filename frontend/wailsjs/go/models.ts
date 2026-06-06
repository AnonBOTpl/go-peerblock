export namespace config {
	
	export class Config {
	    protection_enabled: boolean;
	    start_minimized: boolean;
	    start_with_system: boolean;
	    worker_count: number;
	    cache_size: number;
	    cache_ttl: number;
	    update_interval: number;
	    log_level: string;
	    log_max_size_mb: number;
	    sources: updater.Source[];
	    allowlist: string[];
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.protection_enabled = source["protection_enabled"];
	        this.start_minimized = source["start_minimized"];
	        this.start_with_system = source["start_with_system"];
	        this.worker_count = source["worker_count"];
	        this.cache_size = source["cache_size"];
	        this.cache_ttl = source["cache_ttl"];
	        this.update_interval = source["update_interval"];
	        this.log_level = source["log_level"];
	        this.log_max_size_mb = source["log_max_size_mb"];
	        this.sources = this.convertValues(source["sources"], updater.Source);
	        this.allowlist = source["allowlist"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace filter {
	
	export class Stats {
	    allowed: number;
	    blocked: number;
	    dropped: number;
	    started_at: number;
	
	    static createFrom(source: any = {}) {
	        return new Stats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.allowed = source["allowed"];
	        this.blocked = source["blocked"];
	        this.dropped = source["dropped"];
	        this.started_at = source["started_at"];
	    }
	}

}

export namespace logger {
	
	export class LogEntry {
	    // Go type: time
	    timestamp: any;
	    level: number;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new LogEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.timestamp = this.convertValues(source["timestamp"], null);
	        this.level = source["level"];
	        this.message = source["message"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace updater {
	
	export class Source {
	    name: string;
	    url: string;
	    format: number;
	    enabled: boolean;
	    // Go type: time
	    last_sync: any;
	
	    static createFrom(source: any = {}) {
	        return new Source(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.url = source["url"];
	        this.format = source["format"];
	        this.enabled = source["enabled"];
	        this.last_sync = this.convertValues(source["last_sync"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

