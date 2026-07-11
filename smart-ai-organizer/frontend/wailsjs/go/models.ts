export namespace main {
	
	export class Config {
	    provider: string;
	    model: string;
	    api_key: string;
	    categories: string[];
	    custom_base_url: string;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.provider = source["provider"];
	        this.model = source["model"];
	        this.api_key = source["api_key"];
	        this.categories = source["categories"];
	        this.custom_base_url = source["custom_base_url"];
	    }
	}
	export class FileInfo {
	    name: string;
	    path: string;
	    extension: string;
	    size_bytes: number;
	    snippet_preview: string;
	    is_image: boolean;
	
	    static createFrom(source: any = {}) {
	        return new FileInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.extension = source["extension"];
	        this.size_bytes = source["size_bytes"];
	        this.snippet_preview = source["snippet_preview"];
	        this.is_image = source["is_image"];
	    }
	}

}

